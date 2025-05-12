package validation

import (
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"
)

func TestChecksumExistsValidator(t *testing.T) {
	tests := map[string]struct {
		dirOpts      []fs.PathOp
		errorMessage string
	}{
		"Validates when checksum.md5 file exists": {
			dirOpts: []fs.PathOp{
				fs.WithDir("metadata",
					fs.WithFile("checksum.md5", ""),
				),
			},
		},
		"Validates when checksum.sha1 file exists": {
			dirOpts: []fs.PathOp{
				fs.WithDir("metadata",
					fs.WithFile("checksum.sha1", ""),
				),
			},
		},
		"Validates when checksum.sha256 file exists": {
			dirOpts: []fs.PathOp{
				fs.WithDir("metadata",
					fs.WithFile("checksum.sha256", ""),
				),
			},
		},
		"Validates when checksum.sha512 file exists": {
			dirOpts: []fs.PathOp{
				fs.WithDir("metadata",
					fs.WithFile("checksum.sha512", ""),
				),
			},
		},
		"Fails when expected files do not exist": {
			dirOpts: []fs.PathOp{
				fs.WithDir("metadata",
					fs.WithFile("notchecksum.txt", ""),
				),
			},
			errorMessage: "transfer does not contain checksums",
		},
	}
	for name, tc := range tests {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			tmpDir := fs.NewDir(t, "transfer", tc.dirOpts...)
			defer tmpDir.Remove()

			validator := ChecksumExistsValidator{path: tmpDir.Path()}
			err := validator.Valid()

			if tc.errorMessage == "" {
				assert.NilError(t, err)
			} else {
				assert.ErrorContains(t, err, tc.errorMessage)
			}
		})
	}
}
