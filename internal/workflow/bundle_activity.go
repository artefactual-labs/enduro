package workflow

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"

	"github.com/artefactual-labs/enduro/internal/amclient/bundler"
	"github.com/mholt/archiver"
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
}

type BundleActivityResult struct {
	Name                string // Name of the transfer.
	Kind                string // Client specific, obtained from name, e.g. "DPJ-SIP".
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
			err = nonRetryableError(err)
		}
	}()

	unar := a.Unarchiver(params.Key, params.TempFile)
	if unar == nil {
		res.FullPath, err = a.SingleFile(ctx, params.TransferDir, params.Key, params.TempFile)
		res.FullPathBeforeStrip = res.FullPath
	} else {
		res.FullPath, res.FullPathBeforeStrip, err = a.Bundle(ctx, unar, params.TransferDir, params.Key, params.TempFile, params.StripTopLevelDir)
	}
	if err != nil {
		return nil, nonRetryableError(err)
	}

	res.RelPath, err = filepath.Rel(params.TransferDir, res.FullPath)
	if err != nil {
		return nil, fmt.Errorf("error calculating relative path to transfer (base=%q, target=%q): %v", params.TransferDir, res.FullPath, err)
	}

	res.Name, res.Kind = a.NameKind(params.Key)

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

	var path = filepath.Join(transferDir, dest.Name())
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

	var tempDirBeforeStrip = tempDir
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
			return "", "", fmt.Errorf("%s: top-level item is not a directory: errPrefix, %s")
		}
		tempDir = filepath.Join(tempDir, fis[0].Name())
	}

	// Delete the archive. We still have a copy in the watched source.
	_ = os.Remove(tempFile)

	return tempDir, tempDirBeforeStrip, nil
}

var regex = regexp.MustCompile(`^(?P<kind>.*)[-_](?P<uuid>[a-zA-Z0-9]{8}-[a-zA-Z0-9]{4}-[1-5][a-zA-Z0-9]{3}-[a-zA-Z0-9]{4}-[a-zA-Z0-9]{12})(?P<fileext>\..*)?$`)

// Name the transfer.
func (a *BundleActivity) NameKind(key string) (name, kind string) {
	matches := regex.FindStringSubmatch(key)

	if len := len(matches); len == 0 {
		name = key
	} else if len == 4 {
		name = fmt.Sprintf("%s-%s", matches[1], matches[2][0:13])
		kind = matches[1]
	}

	return name, kind
}
