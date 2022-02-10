package bagit_test

import (
	"context"
	"flag"
	"os/exec"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/artefactual-labs/enduro/internal/bagit"
)

func TestBagitValidationWithGoTool(t *testing.T) {
	ctx := context.Background()
	bagit.UseGoBagit()

	tests := []struct {
		path        string
		errContains string
	}{
		{
			path:        "./tests/test-bagged-transfer",
			errContains: "",
		},
		{
			path:        "./tests/test-bagged-transfer-with-invalid-oxum",
			errContains: "Payload-Oxum validation failed. Expected 1 files and 7 bytes but found 2 files and 7 bytes",
		},
		{
			path:        "./tests/test-bagged-transfer-with-missing-manifest",
			errContains: "Bag validation failed: tests/test-bagged-transfer-with-missing-manifest/manifest-sha256.txt does not exist;",
		},
		{
			path:        "./tests/test-bagged-transfer-with-unexpected-files",
			errContains: "Bag validation failed: data/dos.txt exists on filesystem but is not in the manifest",
		},
		{
			path:        "./tests/test-bagged-transfer-with-invalid-checksums",
			errContains: "Bag validation failed: data/adios.txt sha256 validation failed",
		},
	}
	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			tc := tc
			t.Parallel()

			err := bagit.Valid(ctx, tc.path)
			if tc.errContains == "" {
				assert.NilError(t, err)
			} else {
				assert.ErrorContains(t, err, tc.errContains)
			}
		})
	}
}

var testPythonBagit = flag.Bool("bagit-python", false, "Test with bagit-python (needs to be installed)")

func TestBagitValidationWithPyTool(t *testing.T) {
	if _, err := exec.LookPath("bagit.py"); err != nil {
		t.Skip("bagit.py not installed4")
	}

	ctx := context.Background()
	bagit.UsePyBagit()

	tests := []struct {
		path        string
		errContains string
	}{
		{
			path:        "./tests/test-bagged-transfer",
			errContains: "",
		},
		{
			path:        "./tests/test-bagged-transfer-with-invalid-oxum",
			errContains: "exit status 1",
		},
		{
			path:        "./tests/test-bagged-transfer-with-missing-manifest",
			errContains: "exit status 1",
		},
		{
			path:        "./tests/test-bagged-transfer-with-unexpected-files",
			errContains: "exit status 1",
		},
		{
			path:        "./tests/test-bagged-transfer-with-invalid-checksums",
			errContains: "exit status 1",
		},
	}
	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			tc := tc
			t.Parallel()

			err := bagit.Valid(ctx, tc.path)
			if tc.errContains == "" {
				assert.NilError(t, err)
			} else {
				assert.ErrorContains(t, err, tc.errContains)
			}
		})
	}
}
