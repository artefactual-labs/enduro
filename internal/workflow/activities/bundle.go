package activities

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/otiai10/copy"

	"github.com/artefactual-labs/enduro/internal/bagit"
	"github.com/artefactual-labs/enduro/internal/bundler"
	"github.com/artefactual-labs/enduro/internal/temporal"
)

type BundleActivity struct{}

func NewBundleActivity() *BundleActivity {
	return &BundleActivity{}
}

type BundleActivityParams struct {
	TransferDir        string
	Key                string
	TempFile           string
	StripTopLevelDir   bool
	ExcludeHiddenFiles bool
	IsDir              bool
	BatchDir           string
	Unbag              bool
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
			err = temporal.NewNonRetryableError(err)
		}
	}()

	if params.BatchDir != "" {
		var batchDirIsInTransferDir bool
		batchDirIsInTransferDir, err = isSubPath(params.TransferDir, params.BatchDir)
		if err != nil {
			return nil, temporal.NewNonRetryableError(err)
		}
		if batchDirIsInTransferDir {
			res.FullPath = filepath.Join(params.BatchDir, params.Key)
			// This makes the workflow not to delete the original content in the transfer directory
			res.FullPathBeforeStrip = ""
			if params.ExcludeHiddenFiles {
				if err := removeHiddenFiles(res.FullPath); err != nil {
					return nil, temporal.NewNonRetryableError(fmt.Errorf("failed to remove hidden files: %w", err))
				}
			}
		} else {
			src := filepath.Join(params.BatchDir, params.Key)
			dst := params.TransferDir
			res.FullPath, res.FullPathBeforeStrip, err = a.Copy(ctx, src, dst, params.StripTopLevelDir, params.ExcludeHiddenFiles)
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

	res.RelPath, err = filepath.Rel(params.TransferDir, res.FullPath)
	if err != nil {
		return nil, fmt.Errorf("error calculating relative path to transfer (base=%q, target=%q): %v", params.TransferDir, res.FullPath, err)
	}

	return res, err
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

// Copy a transfer in the given destination using an intermediate temp. directory.
func (a *BundleActivity) Copy(ctx context.Context, src, dst string, stripTopLevelDir bool, excludeHiddenFiles bool) (string, string, error) {
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

func removeHiddenFiles(path string) error {
	return filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if strings.HasPrefix(info.Name(), ".") {
			return os.Remove(path)
		}
		return nil
	})
}
