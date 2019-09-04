package bundler

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/afero"
)

const fileMode = os.FileMode(0o755)

// Bundler helps create Archivematica transfers in the filesystem.
//
// It is a simpler alternative to amclient.TransferSession that does not concern
// with the submission of the transfer.
type Bundler struct {
	fs afero.Fs

	metadata        *MetadataSet
	checksumsMD5    *ChecksumSet
	checksumsSHA1   *ChecksumSet
	checksumsSHA256 *ChecksumSet
}

// NewBundler returns a new Bundler.
func NewBundler(fs afero.Fs) (*Bundler, error) {
	ok, err := afero.DirExists(fs, "")
	if err != nil {
		return nil, fmt.Errorf("error creating bundler: fs check failed, %w", err)
	}
	if !ok {
		return nil, errors.New("error creating bundler: fs does not exist")
	}

	b := &Bundler{
		fs: fs,
	}

	if err := b.createInternalDirs(); err != nil {
		return nil, fmt.Errorf("error creating bundler: %w", err)
	}

	b.metadata = NewMetadataSet(b.fs)
	b.checksumsMD5 = NewChecksumSet("md5", b.fs)
	b.checksumsSHA1 = NewChecksumSet("sha1", b.fs)
	b.checksumsSHA256 = NewChecksumSet("sha256", b.fs)

	return b, nil
}

// NewBundlerWithTempDir returns a bundler based on a temporary directory
// created under the path given.
func NewBundlerWithTempDir(path string) (*Bundler, error) {
	var mode = os.FileMode(0o755)
	var osFs = afero.NewOsFs()
	var baseFs = afero.NewBasePathFs(osFs, path)

	const containerDir = "c"
	ok, err := afero.DirExists(baseFs, containerDir)
	if err != nil {
		return nil, fmt.Errorf("error creating bundler: %v", err)
	}
	if !ok {
		if err := baseFs.Mkdir(containerDir, mode); err != nil {
			return nil, fmt.Errorf("error creating bundler: %v", err)
		}
	}

	tmpdir, err := afero.TempDir(baseFs, containerDir, "")
	if err != nil {
		return nil, fmt.Errorf("error creating bundler: %v", err)
	}

	if err := baseFs.Chmod(tmpdir, mode); err != nil {
		return nil, fmt.Errorf("error creating bundler: %v", err)
	}

	return NewBundler(
		afero.NewBasePathFs(baseFs, tmpdir),
	)
}

// Create a file and return it.
func (b *Bundler) Create(name string) (afero.File, error) {
	err := b.fs.MkdirAll(filepath.Dir(name), fileMode)
	if err != nil {
		return nil, fmt.Errorf("error creating file: %w", err)
	}

	file, err := b.fs.Create(name)
	if err != nil {
		return nil, fmt.Errorf("error creating file: %w", err)
	}

	return file, nil
}

// Write a file with the contents in a given io.Reader.
func (b *Bundler) Write(name string, r io.Reader) error {
	file, err := b.Create(name)
	if err != nil {
		return fmt.Errorf("error writing file: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	if _, err := io.Copy(file, r); err != nil {
		return fmt.Errorf("error writing file: %w", err)
	}

	return nil
}

func (b *Bundler) Bundle() error {
	if err := b.metadata.Write(); err != nil {
		return fmt.Errorf("error during bundling transfer metadata: %w", err)
	}

	if err := b.checksumsMD5.Write(); err != nil {
		return fmt.Errorf("error during bundling MD5 checksums: %w", err)
	}
	if err := b.checksumsSHA1.Write(); err != nil {
		return fmt.Errorf("error during bundling SHA1 checksums: %w", err)
	}
	if err := b.checksumsSHA256.Write(); err != nil {
		return fmt.Errorf("error during bundling SHA256 checksums: %w", err)
	}

	return nil
}

func (b *Bundler) FullBaseFsPath() string {
	bp, ok := b.fs.(*afero.BasePathFs)
	if !ok {
		return ""
	}
	return afero.FullBaseFsPath(bp, "")
}

func (b *Bundler) Destroy() error {
	return b.fs.RemoveAll("")
}

// DescribeFile registers metadata of a file. It causes the transfer to include
// a `metadata.json` file with the metadata of each file described.
func (b *Bundler) DescribeFile(name, field, value string) {
	b.metadata.Add(name, field, value)
}

// Describe registers metadata of the whole dataset/transfer. It causes the
// transfer to include a `metadata.json` file with the metadata included.
func (b *Bundler) Describe(field, value string) {
	b.metadata.Add("objects/", field, value)
}

// ChecksumMD5 registers a MD5 checksum for a file.
func (b *Bundler) ChecksumMD5(name, sum string) {
	b.checksumsMD5.Add(name, sum)
}

// ChecksumSHA1 registers a SHA1 checksum for a file.
func (b *Bundler) ChecksumSHA1(name, sum string) {
	b.checksumsSHA1.Add(name, sum)
}

// ChecksumSHA256 registers a SHA256 checksum for a file.
func (b *Bundler) ChecksumSHA256(name, sum string) {
	b.checksumsSHA256.Add(name, sum)
}

func (b *Bundler) createInternalDirs() error {
	paths := []string{
		"/metadata",
		"/objects",
	}
	for _, path := range paths {
		err := b.fs.Mkdir(path, fileMode)
		if err != nil {
			return fmt.Errorf("error creating internal directory %s: %w", path, err)
		}
	}
	return nil
}
