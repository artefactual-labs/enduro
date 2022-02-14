package bagit_test

import (
	"testing"

	"gotest.tools/v3/assert"

	"github.com/artefactual-labs/enduro/internal/bagit"
)

func TestBagit(t *testing.T) {
	assert.NilError(t, bagit.Complete("./tests/test-bagged-transfer"))
	assert.ErrorContains(t, bagit.Complete("./tests/test-bagged-transfer-with-invalid-oxum"), "Payload-Oxum validation failed. Expected 1 files and 7 bytes but found 2 files and 7 bytes")
	assert.ErrorContains(t, bagit.Complete("./tests/test-bagged-transfer-with-missing-manifest"), "Bag validation failed: tests/test-bagged-transfer-with-missing-manifest/manifest-sha256.txt does not exist")
	assert.ErrorContains(t, bagit.Complete("./tests/test-bagged-transfer-with-unexpected-files"), "Bag validation failed: data/dos.txt exists on filesystem but is not in the manifest")
	assert.ErrorContains(t, bagit.Complete("./tests/test-bagged-transfer-with-invalid-checksums"), "Bag validation failed: data/adios.txt sha256 validation failed")
}
