package activities

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/mholt/archiver/v3"
	"github.com/otiai10/copy"

	"github.com/artefactual-labs/enduro/internal/amclient/bundler"
	"github.com/artefactual-labs/enduro/internal/bagit"
	"github.com/artefactual-labs/enduro/internal/watcher"
	wferrors "github.com/artefactual-labs/enduro/internal/workflow/errors"
	"github.com/artefactual-labs/enduro/internal/workflow/manager"
)

type BundleActivity struct {
	manager *manager.Manager
}

func NewBundleActivity(m *manager.Manager) *BundleActivity {
	return &BundleActivity{manager: m}
}

type BundleActivityParams struct {
	WatcherName      string
	TransferDir      string
	Key              string
	TempFile         string
	StripTopLevelDir bool
	IsDir            bool
	BatchDir         string
}

type BundleActivityResult struct {
	RelPath             string // Path of the transfer relative to the transfer directory.
	FullPath            string // Full path to the transfer in the worker running the session.
	FullPathBeforeStrip string // Same as FullPath but includes the top-level dir even when stripped.
}

func (a *BundleActivity) Execute(ctx context.Context, params *BundleActivityParams) (*BundleActivityResult, error) {
	var (
		res = &BundleActivityResult{}
		err error
	)

	defer func() {
		if err != nil {
			err = wferrors.NonRetryableError(err)
		}
	}()

	if params.BatchDir != "" {
		var batchDirIsInTransferDir bool
		batchDirIsInTransferDir, err = isSubPath(params.TransferDir, params.BatchDir)
		if err != nil {
			return nil, wferrors.NonRetryableError(err)
		}
		if batchDirIsInTransferDir {
			res.FullPath = filepath.Join(params.BatchDir, params.Key)
			// This makes the workflow not to delete the original content in the transfer directory
			res.FullPathBeforeStrip = ""
		} else {
			src := filepath.Join(params.BatchDir, params.Key)
			dst := params.TransferDir
			res.FullPath, res.FullPathBeforeStrip, err = a.Copy(ctx, src, dst, params.StripTopLevelDir)
		}
	} else if params.IsDir {
		var w watcher.Watcher
		w, err = a.manager.Watcher.ByName(params.WatcherName)
		if err == nil {
			src := filepath.Join(w.Path(), params.Key)
			dst := params.TransferDir
			res.FullPath, res.FullPathBeforeStrip, err = a.Copy(ctx, src, dst, false)
		}
	} else {
		unar := a.Unarchiver(params.Key, params.TempFile)
		if unar == nil {
			res.FullPath, err = a.SingleFile(ctx, params.TransferDir, params.Key, params.TempFile)
			res.FullPathBeforeStrip = res.FullPath
		} else {
			res.FullPath, res.FullPathBeforeStrip, err = a.Bundle(ctx, unar, params.TransferDir, params.Key, params.TempFile, params.StripTopLevelDir)
		}
	}
	if err != nil {
		return nil, wferrors.NonRetryableError(err)
	}

	err = unbag(res.FullPath)
	if err != nil {
		return nil, wferrors.NonRetryableError(err)
	}

	res.RelPath, err = filepath.Rel(params.TransferDir, res.FullPath)
	if err != nil {
		return nil, fmt.Errorf("error calculating relative path to transfer (base=%q, target=%q): %v", params.TransferDir, res.FullPath, err)
	}

	return res, err
}

// Unarchiver returns the unarchiver suited for the archival format.
func (a *BundleActivity) Unarchiver(key, filename string) archiver.Unarchiver {
	if iface, err := archiver.ByExtension(key); err == nil {
		if u, ok := iface.(archiver.Unarchiver); ok {
			return u
		}
	}

	file, err := os.Open(filename)
	if err != nil {
		return nil
	}
	defer file.Close()
	if u, err := archiver.ByHeader(file); err == nil {
		return u
	}

	return nil
}

// SingleFile bundles a transfer with the downloaded blob in it.
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

// Bundle a transfer with the contents found in the archive.
func (a *BundleActivity) Bundle(ctx context.Context, unar archiver.Unarchiver, transferDir, key, tempFile string, stripTopLevelDir bool) (string, string, error) {
	// Create a new directory for our transfer with the name randomized.
	const prefix = "enduro"
	tempDir, err := ioutil.TempDir(transferDir, prefix)
	if err != nil {
		return "", "", fmt.Errorf("error creating temporary directory: %s", err)
	}
	_ = os.Chmod(tempDir, os.FileMode(0o755))

	if err := unar.Unarchive(tempFile, tempDir); err != nil {
		return "", "", fmt.Errorf("error unarchiving file: %v", err)
	}

	tempDirBeforeStrip := tempDir
	if stripTopLevelDir {
		tempDir, err = stripDirContainer(tempDir)
		if err != nil {
			return "", "", err
		}
	}

	// Delete the archive. We still have a copy in the watched source.
	_ = os.Remove(tempFile)

	return tempDir, tempDirBeforeStrip, nil
}

// Copy a transfer in the given destination using an intermediate temp. directory.
func (a *BundleActivity) Copy(ctx context.Context, src, dst string, stripTopLevelDir bool) (string, string, error) {
	const prefix = "enduro"
	tempDir, err := ioutil.TempDir(dst, prefix)
	if err != nil {
		return "", "", fmt.Errorf("error creating temporary directory: %s", err)
	}
	_ = os.Chmod(tempDir, os.FileMode(0o755))

	if err := copy.Copy(src, tempDir); err != nil {
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

func isSubPath(path, subPath string) (bool, error) {
	up := ".." + string(os.PathSeparator)
	rel, err := filepath.Rel(path, subPath)
	if err != nil {
		return false, err
	}
	if !strings.HasPrefix(rel, up) && rel != ".." {
		return true, nil
	}
	return false, nil
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
