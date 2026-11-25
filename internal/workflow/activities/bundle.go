package activities

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	bagit "github.com/artefactual-labs/bagit-gython"
	"github.com/otiai10/copy"

	legacybagit "github.com/artefactual-labs/enduro/internal/bagit"
	"github.com/artefactual-labs/enduro/internal/bundler"
	"github.com/artefactual-labs/enduro/internal/temporal"
	"github.com/artefactual-labs/enduro/internal/watcher"
)

// BundleActivity prepares transfer content for an Archivematica pipeline run.
//
// The activity normalizes three source shapes into a transfer that later
// workflow steps can address with one BundleActivityResult:
//   - a downloaded single file is moved into a new bundled transfer under
//     TransferDir;
//   - a watched or extracted directory is copied into an enduro* temporary
//     directory under TransferDir, optionally stripping one top-level directory;
//   - a batch transfer is either reused in place or copied into TransferDir,
//     depending on whether filepath.Join(BatchDir, Key) already refers to
//     content under TransferDir.
//
// Batch containment is intentionally filesystem-aware. A batch path may not be
// textually below TransferDir while still pointing into it through a symlink,
// bind mount, or other filesystem alias. In that case the activity reuses the
// original transfer unless bag conversion needs a private copy for retries.
//
// Reusing in-place batch transfers also changes cleanup semantics:
// FullPathBeforeStrip is left empty so later workflow steps do not remove
// original content that Enduro did not create. Copied and bundled transfers set
// FullPathBeforeStrip to the temporary path that cleanup may remove later.
//
// After staging or reuse, the activity may remove hidden files and optionally
// convert BagIt packages into Archivematica's standard transfer layout.
type BundleActivity struct {
	watchers watcher.Service
	bags     *bagit.Validator
}

// NewBundleActivity creates a bundle activity instance.
func NewBundleActivity(watchers watcher.Service, bags *bagit.Validator) *BundleActivity {
	return &BundleActivity{watchers: watchers, bags: bags}
}

// BundleActivityParams configures how transfer content should be staged.
type BundleActivityParams struct {
	WatcherName         string `json:",omitempty"` // Watcher that supplied an unarchived directory.
	TransferDir         string // Pipeline transfer source directory.
	Key                 string // Object key, batch transfer name, or destination file name.
	TempFile            string // Downloaded file or extracted directory to stage for non-batch transfers.
	StripTopLevelDir    bool   // Remove the copied directory wrapper when it has exactly one child directory.
	ExcludeHiddenFiles  bool   // Remove or skip dotfiles and dot-directories from the staged transfer.
	IsDir               bool   // Treat TempFile as a directory transfer instead of a single file.
	BatchDir            string // Watched batch directory containing Key when processing a batch transfer.
	Unbag               bool   // Convert a BagIt package into an Archivematica transfer after staging.
	CheckBagBeforeUnbag bool   // Check completeness before conversion on a new submission.
	RequireBag          bool   // The submission explicitly requires an unzipped bag.
}

// BundleActivityResult identifies the transfer location after staging.
type BundleActivityResult struct {
	RelPath             string                       // Path of the transfer relative to the transfer directory.
	FullPath            string                       // Full path to the transfer in the worker running the session.
	Layout              *legacybagit.ConvertedLayout `json:",omitempty"`
	Unbagged            bool                         `json:",omitempty"` // The staged content was converted to a standard transfer.
	FullPathBeforeStrip string                       // Same as FullPath but includes the top-level dir even when stripped.
}

// Execute stages or reuses transfer content and returns its pipeline-visible path.
func (a *BundleActivity) Execute(ctx context.Context, params *BundleActivityParams) (_ *BundleActivityResult, err error) {
	res := &BundleActivityResult{}
	// On failure the workflow receives no bundle result to clean up. Only
	// private staging directories have an owned path here.
	defer func() {
		if err == nil || res.FullPathBeforeStrip == "" {
			return
		}
		if cleanupErr := os.RemoveAll(res.FullPathBeforeStrip); cleanupErr != nil {
			combined := errors.Join(err, fmt.Errorf("clean up staged transfer: %w", cleanupErr))
			if temporal.NonRetryableError(err) {
				combined = temporal.NewNonRetryableError(combined)
			}
			err = combined
		}
	}()

	if params.BatchDir != "" {
		src := filepath.Join(params.BatchDir, params.Key)
		if params.IsDir {
			var batchPathIsInTransferDir bool
			res.RelPath, batchPathIsInTransferDir, err = relPathIfUnder(params.TransferDir, src)
			if err != nil {
				return nil, temporal.NewNonRetryableError(err)
			}
			convertInPlaceBag := false
			if batchPathIsInTransferDir && params.Unbag {
				_, err := os.Lstat(filepath.Join(src, "bagit.txt"))
				if err != nil && !errors.Is(err, os.ErrNotExist) {
					return nil, err
				}
				convertInPlaceBag = err == nil
			}
			if batchPathIsInTransferDir && !convertInPlaceBag {
				res.FullPath = src
				// Reused batch content is original transfer content, so leave
				// FullPathBeforeStrip empty to keep later cleanup from removing it.
				res.FullPathBeforeStrip = ""
				if params.ExcludeHiddenFiles {
					if err := removeHiddenFiles(res.FullPath); err != nil {
						return nil, temporal.NewNonRetryableError(fmt.Errorf("failed to remove hidden files: %w", err))
					}
				}
			} else {
				// Already-contained sources are transfer roots, without a wrapper.
				res.RelPath = ""
				strip := params.StripTopLevelDir && !batchPathIsInTransferDir
				res.FullPath, res.FullPathBeforeStrip, err = a.Copy(ctx, src, params.TransferDir, strip, params.ExcludeHiddenFiles)
			}
		} else {
			res.FullPath, err = a.CopySingleFile(params.TransferDir, params.Key, src)
			res.FullPathBeforeStrip = res.FullPath
		}
	} else if params.IsDir && params.WatcherName != "" && params.TempFile == "" {
		source, openErr := a.openWatchedDirectory(params.WatcherName, params.Key)
		if openErr != nil {
			return nil, openErr
		}
		defer source.Close()
		res.FullPath, res.FullPathBeforeStrip, err = copyDirectory(ctx, source, params.TransferDir, params.StripTopLevelDir, params.ExcludeHiddenFiles)
		if err == nil {
			staged, err := os.OpenRoot(res.FullPath)
			if err != nil {
				return nil, err
			}
			defer staged.Close()
			if err := checkTransferLinks(ctx, staged); err != nil {
				return nil, err
			}
		}
	} else if params.IsDir {
		res.FullPath, res.FullPathBeforeStrip, err = a.Copy(ctx, params.TempFile, params.TransferDir, params.StripTopLevelDir, params.ExcludeHiddenFiles)
	} else {
		res.FullPath, err = a.SingleFile(ctx, params.TransferDir, params.Key, params.TempFile)
		res.FullPathBeforeStrip = res.FullPath
	}
	if err != nil {
		return nil, temporal.NewNonRetryableError(err)
	}

	if params.Unbag {
		if params.CheckBagBeforeUnbag {
			// The completeness check needs the bag layout before conversion.
			if _, err := checkStagedBag(ctx, res.FullPath, params.RequireBag, a.bags.CheckCompletenessContext); err != nil {
				return nil, err
			}
		}
		res.Layout, err = unbag(res.FullPath)
		res.Unbagged = res.Layout != nil
		if err != nil {
			return nil, temporal.NewNonRetryableError(err)
		}
	}

	if res.RelPath == "" {
		res.RelPath, err = filepath.Rel(params.TransferDir, res.FullPath)
		if err != nil {
			return nil, fmt.Errorf("error calculating relative path to transfer (base=%q, target=%q): %v", params.TransferDir, res.FullPath, err)
		}
	}

	return res, err
}

func (a *BundleActivity) openWatchedDirectory(watcherName, key string) (*os.Root, error) {
	w, err := a.watchers.ByName(watcherName)
	if err != nil {
		return nil, temporal.NewNonRetryableError(err)
	}
	if w.Path() == "" || !filepath.IsLocal(key) || key == "." {
		return nil, temporal.NewNonRetryableError(fmt.Errorf("invalid directory %q for filesystem watcher %q", key, watcherName))
	}
	watched, err := os.OpenRoot(w.Path())
	if err != nil {
		return nil, fmt.Errorf("open watcher directory: %w", err)
	}
	defer watched.Close()
	source, err := watched.OpenRoot(key)
	if err != nil {
		return nil, temporal.NewNonRetryableError(fmt.Errorf("open transfer in watcher directory: %w", err))
	}
	return source, nil
}

// SingleFile creates a transfer bundle containing the downloaded blob.
func (a *BundleActivity) SingleFile(ctx context.Context, transferDir, key, tempFile string) (string, error) {
	b, err := bundler.NewBundlerWithTempDir(transferDir)
	if err != nil {
		return "", fmt.Errorf("error creating bundle: %v", err)
	}

	dest, err := b.Create(filepath.Join("objects", key))
	if err != nil {
		return "", fmt.Errorf("error creating file: %v", err)
	}
	defer dest.Close()

	path := filepath.Join(transferDir, dest.Name())
	if err := os.Rename(tempFile, path); err != nil {
		return "", fmt.Errorf("error moving file (from %s to %s): %v", tempFile, path, err)
	}

	if err := os.Chmod(path, os.FileMode(0o755)); err != nil {
		return "", fmt.Errorf("error changing file mode: %v", err)
	}

	if err := b.Bundle(); err != nil {
		return "", fmt.Errorf("error bundling the transfer: %v", err)
	}

	return b.FullBaseFsPath(), nil
}

// CopySingleFile creates a transfer bundle containing a source file without
// moving the original file.
func (a *BundleActivity) CopySingleFile(transferDir, key, src string) (string, error) {
	b, err := bundler.NewBundlerWithTempDir(transferDir)
	if err != nil {
		return "", fmt.Errorf("error creating bundle: %v", err)
	}

	source, err := os.Open(src)
	if err != nil {
		return "", fmt.Errorf("error opening file: %v", err)
	}
	defer source.Close()

	dest, err := b.Create(filepath.Join("objects", key))
	if err != nil {
		return "", fmt.Errorf("error creating file: %v", err)
	}

	destPath := filepath.Join(transferDir, dest.Name())
	if _, err := io.Copy(dest, source); err != nil {
		_ = dest.Close()
		return "", fmt.Errorf("error copying file: %v", err)
	}
	if err := dest.Close(); err != nil {
		return "", fmt.Errorf("error closing file: %v", err)
	}

	if err := os.Chmod(destPath, os.FileMode(0o755)); err != nil {
		return "", fmt.Errorf("error changing file mode: %v", err)
	}

	if err := b.Bundle(); err != nil {
		return "", fmt.Errorf("error bundling the transfer: %v", err)
	}

	return b.FullBaseFsPath(), nil
}

// Copy copies a transfer into dst using an intermediate temporary directory.
//
// It returns the final transfer path and the path before StripTopLevelDir was
// applied. When excludeHiddenFiles is enabled, dotfiles and dot-directories are
// skipped during the copy.
func (a *BundleActivity) Copy(ctx context.Context, src, dst string, stripTopLevelDir, excludeHiddenFiles bool) (string, string, error) {
	if src == "" {
		return "", "", errors.New("missing transfer source directory")
	}
	// A transfer directory may itself be a symlink. Stage its contents so
	// filtering and conversion operate on a private directory.
	source, err := os.OpenRoot(src)
	if err != nil {
		return "", "", fmt.Errorf("open transfer source directory: %w", err)
	}
	defer source.Close()
	return copyDirectory(ctx, source, dst, stripTopLevelDir, excludeHiddenFiles)
}

func copyDirectory(ctx context.Context, source *os.Root, dst string, stripTopLevelDir, excludeHiddenFiles bool) (string, string, error) {
	const prefix = "enduro"
	tempDir, err := os.MkdirTemp(dst, prefix)
	if err != nil {
		return "", "", fmt.Errorf("error creating temporary directory: %s", err)
	}
	_ = os.Chmod(tempDir, os.FileMode(0o755))

	stagingDir, err := os.Stat(tempDir)
	if err != nil {
		return "", "", errors.Join(err, os.RemoveAll(tempDir))
	}
	if err := copy.Copy(".", tempDir, copy.Options{
		FS: source.FS(),
		Skip: func(srcinfo os.FileInfo, src, dest string) (bool, error) {
			if err := ctx.Err(); err != nil {
				return false, err
			}
			// Catch transfer directories nested under the source, including
			// filesystem aliases, before following our own staging copy.
			if srcinfo.IsDir() && os.SameFile(srcinfo, stagingDir) {
				return false, errors.New("cannot stage a transfer inside its source directory")
			}
			// Exclude hidden files.
			if excludeHiddenFiles && strings.HasPrefix(srcinfo.Name(), ".") {
				return true, nil
			}
			// copy's default symlink handling reads from the working directory
			// even with an FS set. Read links through the open transfer root.
			if srcinfo.Mode()&os.ModeSymlink != 0 {
				target, err := source.Readlink(src)
				if err != nil {
					return false, err
				}
				return true, os.Symlink(target, dest)
			}
			return false, nil
		},
	}); err != nil {
		return "", "", errors.Join(fmt.Errorf("error copying transfer: %w", err), os.RemoveAll(tempDir))
	}

	tempDirBeforeStrip := tempDir
	if stripTopLevelDir {
		tempDir, err = stripDirContainer(tempDir)
		if err != nil {
			return "", "", errors.Join(err, os.RemoveAll(tempDirBeforeStrip))
		}
	}

	return tempDir, tempDirBeforeStrip, nil
}

// relPathIfUnder returns target's relative path when target is already under base.
//
// The check intentionally goes beyond filepath.Rel. Batch directories may be
// configured through symlinks or mount aliases that point into TransferDir, and
// those should be reused instead of copied into a new temporary directory.
func relPathIfUnder(base, target string) (string, bool, error) {
	baseAbs, err := filepath.Abs(base)
	if err != nil {
		return "", false, err
	}
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return "", false, err
	}

	if rel, ok, err := lexicalRelPathIfUnder(baseAbs, targetAbs); err != nil || ok {
		return rel, ok, err
	}

	if rel, ok, err := evaluatedRelPathIfUnder(baseAbs, targetAbs); err != nil || ok {
		return rel, ok, err
	}

	return sameFileRelPathIfUnder(baseAbs, targetAbs)
}

// evaluatedRelPathIfUnder repeats the descendant check after resolving symlinks.
func evaluatedRelPathIfUnder(base, target string) (string, bool, error) {
	baseEval, err := filepath.EvalSymlinks(base)
	if err != nil {
		return "", false, err
	}
	targetEval, err := filepath.EvalSymlinks(target)
	if err != nil {
		return "", false, err
	}
	return lexicalRelPathIfUnder(baseEval, targetEval)
}

type sameFileCandidate struct {
	info   os.FileInfo
	suffix []string
}

// sameFileRelPathIfUnder detects filesystem aliases that path evaluation misses.
//
// Symlink evaluation does not resolve bind mounts. This fallback compares
// directory identities with os.SameFile so a batch path reached through a mount
// alias can still be recognized as content already under TransferDir.
func sameFileRelPathIfUnder(baseAbs, targetAbs string) (string, bool, error) {
	baseInfo, err := os.Stat(baseAbs)
	if err != nil {
		return "", false, err
	}

	var candidates []sameFileCandidate
	var relParts []string
	for current := targetAbs; ; current = filepath.Dir(current) {
		currentInfo, err := os.Stat(current)
		if err != nil {
			return "", false, err
		}
		if os.SameFile(baseInfo, currentInfo) {
			if len(relParts) == 0 {
				return ".", true, nil
			}
			return filepath.Join(relParts...), true, nil
		}
		if currentInfo.IsDir() {
			candidates = append(candidates, sameFileCandidate{
				info:   currentInfo,
				suffix: append([]string(nil), relParts...),
			})
		}

		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		relParts = append([]string{filepath.Base(current)}, relParts...)
	}

	return relPathForSameFileDescendant(baseAbs, candidates)
}

// relPathForSameFileDescendant finds which transfer descendant an alias maps to.
//
// candidates are ancestors from the target path. When one has the same
// filesystem identity as a directory under baseAbs, its stored suffix rebuilds
// the full transfer-relative path below that matching descendant.
func relPathForSameFileDescendant(baseAbs string, candidates []sameFileCandidate) (string, bool, error) {
	if len(candidates) == 0 {
		return "", false, nil
	}

	var rel string
	var ok bool
	err := filepath.WalkDir(baseAbs, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if path == baseAbs {
				return err
			}
			// This walk is only a best-effort alias check. Existing transfer
			// contents may be unreadable or removed concurrently, and that
			// should not prevent an unrelated external batch from being copied.
			return nil
		}
		if !d.IsDir() {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			if path == baseAbs {
				return err
			}
			return fs.SkipDir
		}
		for _, candidate := range candidates {
			if !os.SameFile(candidate.info, info) {
				continue
			}

			rel, err = filepath.Rel(baseAbs, path)
			if err != nil {
				return err
			}
			rel = relPathWithSuffix(rel, candidate.suffix)
			ok = true
			return fs.SkipAll
		}

		return nil
	})
	if err != nil {
		return "", false, err
	}
	return rel, ok, nil
}

// relPathWithSuffix appends target path components to a transfer-relative path.
func relPathWithSuffix(rel string, suffix []string) string {
	if len(suffix) == 0 {
		return rel
	}
	if rel == "." {
		return filepath.Join(suffix...)
	}

	parts := append([]string{rel}, suffix...)
	return filepath.Join(parts...)
}

// lexicalRelPathIfUnder checks descendant paths without resolving filesystem aliases.
func lexicalRelPathIfUnder(base, target string) (string, bool, error) {
	up := ".." + string(os.PathSeparator)
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return "", false, err
	}
	if rel == ".." || strings.HasPrefix(rel, up) {
		return "", false, nil
	}
	return rel, true, nil
}

// stripDirContainer strips the top-level directory of a transfer.
func stripDirContainer(path string) (string, error) {
	const errPrefix = "error stripping top-level dir"
	ff, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("%s: cannot open path: %v", errPrefix, err)
	}
	defer ff.Close()
	fis, err := ff.Readdir(2)
	if err != nil {
		return "", fmt.Errorf("%s: error reading dir: %v", errPrefix, err)
	}
	if len(fis) != 1 {
		return "", fmt.Errorf("%s: directory has more than one child", errPrefix)
	}
	if !fis[0].IsDir() {
		return "", fmt.Errorf("%s: top-level item is not a directory", errPrefix)
	}
	return filepath.Join(path, fis[0].Name()), nil
}

// unbag converts a bagged transfer into a standard Archivematica transfer.
// The result records whether conversion took place for the submission activity.
func unbag(path string) (*legacybagit.ConvertedLayout, error) {
	root, err := os.OpenRoot(path)
	if err != nil {
		return nil, err
	}
	defer root.Close()

	if _, err := root.Stat("bagit.txt"); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	if err := legacybagit.Complete(path); err != nil {
		return nil, err
	}
	layout := &legacybagit.ConvertedLayout{}
	if err := convertBag(root, layout); err != nil {
		return nil, err
	}
	return layout, nil
}

func convertBag(root *os.Root, layout *legacybagit.ConvertedLayout) error {
	// Preserve a supplied Archivematica transfer layout. Flat bags need an
	// objects directory, with any transfer metadata alongside it.
	isDir := func(name string) (bool, error) {
		info, err := root.Stat(name)
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return err == nil && info.IsDir(), err
	}
	structured, err := isDir("data/objects")
	if err != nil {
		return err
	}
	hasMetadata, err := isDir("data/metadata")
	if err != nil {
		return err
	}
	transferPath := func(payload string) string {
		if structured || (hasMetadata && (payload == "metadata" || strings.HasPrefix(payload, "metadata/"))) {
			return payload
		}
		return "objects/" + payload
	}

	// Track each payload link's destination before moving its containing
	// directory. A link may still resolve after conversion but point to a
	// different file. Conversion always operates on a private staging copy.
	payloadRoot, err := root.OpenRoot("data")
	if err != nil {
		return err
	}
	defer payloadRoot.Close()
	links := make(map[string]os.FileInfo)
	if err := fs.WalkDir(payloadRoot.FS(), ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil || entry.Type()&fs.ModeSymlink == 0 {
			return err
		}
		target, err := payloadRoot.Stat(name)
		if err != nil {
			return fmt.Errorf("resolve payload link %q: %w", name, err)
		}
		links[name] = target
		return nil
	}); err != nil {
		return err
	}
	if structured {
		items, err := fs.ReadDir(root.FS(), "data")
		if err != nil {
			return err
		}
		for _, item := range items {
			layout.PayloadRoots = append(layout.PayloadRoots, item.Name())
			if err := root.Rename(filepath.Join("data", item.Name()), item.Name()); err != nil {
				return err
			}
		}
		if err := root.Remove("data"); err != nil {
			return err
		}
	} else {
		layout.PayloadRoots = []string{"objects"}
		if hasMetadata {
			layout.PayloadRoots = append(layout.PayloadRoots, "metadata")
			if err := root.Rename("data/metadata", "metadata"); err != nil {
				return err
			}
		}
		if err := root.Rename("data", "objects"); err != nil {
			return err
		}
	}

	const metadataPath = "metadata"
	documentationPath := filepath.Join(metadataPath, "submissionDocumentation")
	if err := root.MkdirAll(documentationPath, 0o775); err != nil {
		return err
	}

	// Write manifest checksums to checksum file.
	for _, item := range [][2]string{
		{"manifest-sha512.txt", "checksum.sha512"},
		{"manifest-sha256.txt", "checksum.sha256"},
		{"manifest-sha1.txt", "checksum.sha1"},
		{"manifest-md5.txt", "checksum.md5"},
	} {
		file, err := root.Open(item[0])
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return err
		}
		defer file.Close()

		newFile, err := root.Create(filepath.Join(metadataPath, item[1]))
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return err
		}
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			before, payload, ok := strings.Cut(scanner.Text(), "data/")
			if !ok {
				newFile.Close()
				return errors.New("invalid payload manifest entry")
			}
			if _, err := fmt.Fprintln(newFile, before+"../"+transferPath(payload)); err != nil {
				newFile.Close()
				return err
			}
		}
		if err := errors.Join(scanner.Err(), newFile.Close()); err != nil {
			return err
		}

		break // One file is enough.
	}

	// Move bag files to submissionDocumentation.
	for _, item := range []string{
		"bag-info.txt",
		"bagit.txt",
		"manifest-md5.txt",
		"tagmanifest-md5.txt",
		"manifest-sha1.txt",
		"tagmanifest-sha1.txt",
		"manifest-sha256.txt",
		"tagmanifest-sha256.txt",
		"manifest-sha512.txt",
		"tagmanifest-sha512.txt",
	} {
		if err := root.Rename(item, filepath.Join(documentationPath, item)); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}

	for name, original := range links {
		current, err := root.Stat(transferPath(name))
		if err != nil || !os.SameFile(original, current) {
			return fmt.Errorf("bag conversion changes the target of payload link %q", name)
		}
	}

	return nil
}

// removeHiddenFiles removes dotfiles and dot-directories from path recursively.
func removeHiddenFiles(path string) error {
	root, err := os.OpenRoot(path)
	if err != nil {
		return err
	}
	defer root.Close()

	return fs.WalkDir(root.FS(), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == "." {
			return nil
		}
		if strings.HasPrefix(d.Name(), ".") {
			if d.IsDir() {
				if err := root.RemoveAll(path); err != nil {
					return err
				}
				return fs.SkipDir
			}
			return root.Remove(path)
		}
		return nil
	})
}
