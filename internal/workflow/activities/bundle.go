package activities

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/artefactual-labs/enduro/internal/amclient/bundler"
	wferrors "github.com/artefactual-labs/enduro/internal/workflow/errors"

	"github.com/mholt/archiver/v3"
	"github.com/otiai10/copy"
)

type BundleActivity struct{}

func NewBundleActivity() *BundleActivity {
	return &BundleActivity{}
}

type BundleActivityParams struct {
	TransferDir      string
	Key              string
	TempFile         string
	StripTopLevelDir bool
	BatchDir         string
}

type BundleActivityResult struct {
	RelPath             string // Path of the transfer relative to the transfer directory.
	FullPath            string // Full path to the transfer in the worker running the session.
	FullPathBeforeStrip string // Same as FullPath but includes the top-level dir even when stripped.
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
			res.FullPath, res.FullPathBeforeStrip, err = a.Copy(ctx, params.TransferDir, params.BatchDir, params.Key, params.StripTopLevelDir)
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
		const errPrefix = "error stripping top-level dir"
		ff, err := os.Open(tempDir)
		if err != nil {
			return "", "", fmt.Errorf("%s: error opening dir: %v", errPrefix, err)
		}
		fis, err := ff.Readdir(2)
		if err != nil {
			return "", "", fmt.Errorf("%s: error reading dir: %v", errPrefix, err)
		}
		if len(fis) != 1 {
			return "", "", fmt.Errorf("%s: unexpected number of items were found in the archive", errPrefix)
		}
		if !fis[0].IsDir() {
			return "", "", fmt.Errorf("%s: top-level item is not a directory", errPrefix)
		}
		tempDir = filepath.Join(tempDir, fis[0].Name())
	}

	// Delete the archive. We still have a copy in the watched source.
	_ = os.Remove(tempFile)

	return tempDir, tempDirBeforeStrip, nil
}

func (a *BundleActivity) Copy(ctx context.Context, transferDir, batchDir, key string, stripTopLevelDir bool) (string, string, error) {
	const prefix = "enduro"
	tempDir, err := ioutil.TempDir(transferDir, prefix)
	if err != nil {
		return "", "", fmt.Errorf("error creating temporary directory: %s", err)
	}
	_ = os.Chmod(tempDir, os.FileMode(0o755))

	if err := copy.Copy(filepath.Join(batchDir, key), tempDir); err != nil {
		return "", "", fmt.Errorf("error copying transfer: %v", err)
	}

	tempDirBeforeStrip := tempDir
	if stripTopLevelDir {
		const errPrefix = "error stripping top-level dir"
		ff, err := os.Open(tempDir)
		if err != nil {
			return "", "", fmt.Errorf("%s: error opening dir: %v", errPrefix, err)
		}
		fis, err := ff.Readdir(2)
		if err != nil {
			return "", "", fmt.Errorf("%s: error reading dir: %v", errPrefix, err)
		}
		if len(fis) != 1 {
			return "", "", fmt.Errorf("%s: unexpected number of items were found in the archive", errPrefix)
		}
		if !fis[0].IsDir() {
			return "", "", fmt.Errorf("%s: top-level item is not a directory", errPrefix)
		}
		tempDir = filepath.Join(tempDir, fis[0].Name())
	}

	return tempDir, tempDirBeforeStrip, nil
}
