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

	"github.com/otiai10/copy"

	"github.com/artefactual-labs/enduro/internal/bagit"
	"github.com/artefactual-labs/enduro/internal/bundler"
	"github.com/artefactual-labs/enduro/internal/temporal"
)

// BundleActivity prepares transfer content for an Archivematica pipeline run.
//
// The activity normalizes three source shapes into a transfer that later
// workflow steps can address with one BundleActivityResult:
//   - a downloaded single file is moved into a new bundled transfer under
//     TransferDir;
//   - a downloaded or extracted directory is copied into an enduro* temporary
//     directory under TransferDir, optionally stripping one top-level directory;
//   - a batch transfer is either reused in place or copied into TransferDir,
//     depending on whether filepath.Join(BatchDir, Key) already refers to
//     content under TransferDir.
//
// Batch containment is intentionally filesystem-aware. A batch path may not be
// textually below TransferDir while still pointing into it through a symlink,
// bind mount, or other filesystem alias. In that case the activity reuses the
// original transfer instead of creating an unnecessary staging copy.
//
// Reusing in-place batch transfers also changes cleanup semantics:
// FullPathBeforeStrip is left empty so later workflow steps do not remove
// original content that Enduro did not create. Copied and bundled transfers set
// FullPathBeforeStrip to the temporary path that cleanup may remove later.
//
// After staging or reuse, the activity may remove hidden files and optionally
// convert BagIt packages into Archivematica's standard transfer layout.
type BundleActivity struct{}

// NewBundleActivity creates a bundle activity instance.
func NewBundleActivity() *BundleActivity {
	return &BundleActivity{}
}

// BundleActivityParams configures how transfer content should be staged.
type BundleActivityParams struct {
	TransferDir        string // Pipeline transfer source directory.
	Key                string // Object key, batch transfer name, or destination file name.
	TempFile           string // Downloaded file or extracted directory to stage for non-batch transfers.
	StripTopLevelDir   bool   // Remove the copied directory wrapper when it has exactly one child directory.
	ExcludeHiddenFiles bool   // Remove or skip dotfiles and dot-directories from the staged transfer.
	IsDir              bool   // Treat TempFile as a directory transfer instead of a single file.
	BatchDir           string // Watched batch directory containing Key when processing a batch transfer.
	Unbag              bool   // Convert a BagIt package into an Archivematica transfer after staging.
}

// BundleActivityResult identifies the transfer location after staging.
type BundleActivityResult struct {
	RelPath             string // Path of the transfer relative to the transfer directory.
	FullPath            string // Full path to the transfer in the worker running the session.
	FullPathBeforeStrip string // Same as FullPath but includes the top-level dir even when stripped.
}

// Execute stages or reuses transfer content and returns its pipeline-visible path.
func (a *BundleActivity) Execute(ctx context.Context, params *BundleActivityParams) (*BundleActivityResult, error) {
	var (
		res = &BundleActivityResult{}
		err error
	)

	defer func() {
		if err != nil {
			err = temporal.NewNonRetryableError(err)
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
			if batchPathIsInTransferDir {
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
				dst := params.TransferDir
				res.FullPath, res.FullPathBeforeStrip, err = a.Copy(ctx, src, dst, params.StripTopLevelDir, params.ExcludeHiddenFiles)
			}
		} else {
			res.FullPath, err = a.CopySingleFile(params.TransferDir, params.Key, src)
			res.FullPathBeforeStrip = res.FullPath
		}
	} else if params.IsDir {
		// For a standard (non-batch) package that has been downloaded (and
		// possibly extracted) to a local directory (params.TempFile), copy the
		// package directory to params.TransferDir for processing.
		res.FullPath, res.FullPathBeforeStrip, err = a.Copy(
			ctx,
			params.TempFile,
			params.TransferDir,
			params.StripTopLevelDir,
			params.ExcludeHiddenFiles,
		)
	} else {
		res.FullPath, err = a.SingleFile(ctx, params.TransferDir, params.Key, params.TempFile)
		res.FullPathBeforeStrip = res.FullPath
	}
	if err != nil {
		return nil, temporal.NewNonRetryableError(err)
	}

	if params.Unbag {
		err = unbag(res.FullPath)
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
	const prefix = "enduro"
	tempDir, err := os.MkdirTemp(dst, prefix)
	if err != nil {
		return "", "", fmt.Errorf("error creating temporary directory: %s", err)
	}
	_ = os.Chmod(tempDir, os.FileMode(0o755))

	if err := copy.Copy(src, tempDir, copy.Options{
		Skip: func(srcinfo os.FileInfo, src, dest string) (bool, error) {
			// Exclude hidden files.
			if excludeHiddenFiles && strings.HasPrefix(srcinfo.Name(), ".") {
				return true, nil
			}

			return false, nil
		},
	}); err != nil {
		return "", "", fmt.Errorf("error copying transfer: %v", err)
	}

	tempDirBeforeStrip := tempDir
	if stripTopLevelDir {
		tempDir, err = stripDirContainer(tempDir)
		if err != nil {
			return "", "", err
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
// It returns a nil error if a bag is not identified, and non-nil errors when
// the bag seems invalid, without verifying the actual file contents.
func unbag(path string) error {
	fi, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !fi.IsDir() {
		return errors.New("not a directory")
	}

	// Only continue if we have a bag.
	_, err = os.Stat(filepath.Join(path, "bagit.txt"))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}

	// Confirm completeness of the bag.
	if err := bagit.Complete(path); err != nil {
		return err
	}

	// Move files in data up one level if 'objects' folder already exists.
	// Otherwise, rename data to objects.
	dataPath := filepath.Join(path, "data")
	if fi, err := os.Stat(dataPath); !os.IsNotExist(err) && fi.IsDir() {
		items, err := os.ReadDir(dataPath)
		if err != nil {
			return err
		}
		for _, item := range items {
			src := filepath.Join(dataPath, item.Name())
			dst := filepath.Join(path, filepath.Base(src))
			if err := os.Rename(src, dst); err != nil {
				return err
			}
		}
		if err := os.RemoveAll(dataPath); err != nil {
			return err
		}
	} else {
		dst := filepath.Join(path, "objects")
		if err := os.Rename(dataPath, dst); err != nil {
			return err
		}
	}

	// Create metadata and submissionDocumentation directories.
	metadataPath := filepath.Join(path, "metadata")
	documentationPath := filepath.Join(metadataPath, "submissionDocumentation")
	if err := os.MkdirAll(metadataPath, 0o775); err != nil {
		return err
	}
	if err := os.MkdirAll(documentationPath, 0o775); err != nil {
		return err
	}

	// Write manifest checksums to checksum file.
	for _, item := range [][2]string{
		{"manifest-sha512.txt", "checksum.sha512"},
		{"manifest-sha256.txt", "checksum.sha256"},
		{"manifest-sha1.txt", "checksum.sha1"},
		{"manifest-md5.txt", "checksum.md5"},
	} {
		file, err := os.Open(filepath.Join(path, item[0]))
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return err
		}
		defer file.Close()

		newFile, err := os.Create(filepath.Join(metadataPath, item[1]))
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return err
		}
		defer newFile.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			newLine := ""
			if strings.Contains(line, "data/objects/") {
				newLine = strings.Replace(line, "data/objects/", "../objects/", 1)
			} else {
				newLine = strings.Replace(line, "data/", "../objects/", 1)
			}
			fmt.Fprintln(newFile, newLine)
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
		src := filepath.Join(path, item)
		dst := filepath.Join(documentationPath, item)
		_ = os.Rename(src, dst)
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
