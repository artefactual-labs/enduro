package validation

import (
	"fmt"
	"os"
	"path"

	"github.com/hashicorp/go-multierror"
)

var checksumFiles = [4]string{
	"checksum.md5",
	"checksum.sha1",
	"checksum.sha256",
	"checksum.sha512",
}

func ValidateTransfer(c Config, path string) error {
	var result error

	if c.ChecksumsCheckEnabled {
		v := ChecksumExistsValidator{path: path}
		if err := v.Valid(); err != nil {
			result = multierror.Append(result, err)
		}
	}

	return result
}

// Validator is the interface that all validators must implement.
type Validator interface {
	Valid() error
}

// ChecksumExistsValidator is a Validator that checks...
type ChecksumExistsValidator struct {
	path string
}

func (v ChecksumExistsValidator) Valid() error {
	for _, checksum := range checksumFiles {
		if fileExists(path.Join(v.path, "metadata", checksum)) {
			return nil
		}
	}
	return fmt.Errorf("transfer does not contain checksums (path=%s)", v.path)
}

func fileExists(name string) bool {
	stat, err := os.Stat(name)
	if err != nil {
		return false
	}
	return !stat.IsDir()
}
