package aipstore_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"gocloud.dev/blob"
	"gocloud.dev/blob/memblob"
	"gotest.tools/v3/assert"

	"github.com/artefactual-labs/enduro/internal/aipstore"
)

func TestService(t *testing.T) {
	config := &aipstore.Config{
		Bucket: "bucket string",
		Region: "us-west-1",
		Key:    "key",
		Secret: "secret",
	}

	s, err := aipstore.NewService(config)
	assert.NilError(t, err)

	mb := memblob.OpenBucket(nil)
	aipstore.SetBucket(s, mb)

	file, err := os.CreateTemp("", "test")
	assert.NilError(t, err)

	err = s.UploadAIP(context.Background(), file.Name())
	assert.NilError(t, err)

	iterator := mb.List(&blob.ListOptions{})

	list, err := iterator.Next(context.Background())
	assert.NilError(t, err)
	assert.Equal(t, list.Key, filepath.Base(file.Name()))
}
