package batch

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-logr/logr"
	temporalsdk_mocks "go.temporal.io/sdk/mocks"
	"goa.design/goa/v3/pkg"
	"gotest.tools/v3/assert"

	goabatch "github.com/artefactual-labs/enduro/internal/api/gen/batch"
)

func TestBatchServiceBrowse(t *testing.T) {
	ctx := context.Background()
	logger := logr.Discard()
	client := &temporalsdk_mocks.Client{}

	root := t.TempDir()
	mkdirAll(t, root, "zeta")
	mkdirAll(t, root, "alpha", "child")
	writeFile(t, root, "transfer.zip")

	batchsvc := NewService(logger, client, taskQueue, completedDirs, Config{BrowserRoot: root})

	result, err := batchsvc.Browse(ctx, &goabatch.BrowsePayload{})
	assert.NilError(t, err)
	assert.Equal(t, result.Path, "")
	assert.Equal(t, result.AbsolutePath, root)
	assert.Equal(t, result.Truncated, false)
	assert.Equal(t, len(result.Entries), 2)
	assert.Equal(t, result.Entries[0].Name, "alpha")
	assert.Equal(t, result.Entries[0].Path, "alpha")
	assert.Equal(t, result.Entries[0].AbsolutePath, filepath.Join(root, "alpha"))
	assert.Assert(t, result.Entries[0].ModifiedAt != nil)
	assert.Equal(t, result.Entries[1].Name, "zeta")

	nested := "alpha"
	result, err = batchsvc.Browse(ctx, &goabatch.BrowsePayload{Path: &nested})
	assert.NilError(t, err)
	assert.Equal(t, result.Path, "alpha")
	assert.Equal(t, result.AbsolutePath, filepath.Join(root, "alpha"))
	assert.Equal(t, len(result.Entries), 1)
	assert.Equal(t, result.Entries[0].Path, "alpha/child")
}

func TestBatchServiceBrowseRejectsUnavailableOrInvalidPaths(t *testing.T) {
	ctx := context.Background()
	logger := logr.Discard()
	client := &temporalsdk_mocks.Client{}

	batchsvc := NewService(logger, client, taskQueue, completedDirs)
	_, err := batchsvc.Browse(ctx, &goabatch.BrowsePayload{})
	assertGoaErrorName(t, err, "not_available")

	root := t.TempDir()
	writeFile(t, root, "transfer.zip")
	batchsvc = NewService(logger, client, taskQueue, completedDirs, Config{BrowserRoot: root})

	for _, value := range []string{"../outside", "/tmp", "transfer.zip"} {
		t.Run(value, func(t *testing.T) {
			_, err := batchsvc.Browse(ctx, &goabatch.BrowsePayload{Path: &value})
			assertGoaErrorName(t, err, "not_valid")
		})
	}
}

func TestBatchServiceBrowseTruncatesLargeDirectories(t *testing.T) {
	ctx := context.Background()
	logger := logr.Discard()
	client := &temporalsdk_mocks.Client{}

	root := t.TempDir()
	for i := range browserEntryLimit + 1 {
		mkdirAll(t, root, fmt.Sprintf("transfer-%04d", i))
	}

	batchsvc := NewService(logger, client, taskQueue, completedDirs, Config{BrowserRoot: root})
	result, err := batchsvc.Browse(ctx, &goabatch.BrowsePayload{})

	assert.NilError(t, err)
	assert.Equal(t, result.Truncated, true)
	assert.Equal(t, len(result.Entries), browserEntryLimit)
}

func TestBatchConfigValidate(t *testing.T) {
	t.Run("Normalizes an existing browser root", func(t *testing.T) {
		root := t.TempDir()
		cfg := Config{BrowserRoot: root}

		err := cfg.Validate()

		assert.NilError(t, err)
		assert.Equal(t, filepath.IsAbs(cfg.BrowserRoot), true)
	})

	t.Run("Resolves a relative browser root from a base directory", func(t *testing.T) {
		baseDir := t.TempDir()
		mkdirAll(t, baseDir, "batches")
		cfg := Config{BrowserRoot: "./batches"}

		err := cfg.ValidateWithBaseDir(baseDir)

		assert.NilError(t, err)
		assert.Equal(t, cfg.BrowserRoot, filepath.Join(baseDir, "batches"))
	})

	t.Run("Rejects a non-directory browser root", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, root, "transfer.zip")
		cfg := Config{BrowserRoot: filepath.Join(root, "transfer.zip")}

		err := cfg.Validate()

		assert.ErrorContains(t, err, "is not a directory")
	})
}

func assertGoaErrorName(t *testing.T, err error, name string) {
	t.Helper()

	var goaErr goa.GoaErrorNamer
	assert.Assert(t, errors.As(err, &goaErr))
	assert.Equal(t, goaErr.GoaErrorName(), name)
}

func mkdirAll(t *testing.T, root string, elems ...string) {
	t.Helper()

	err := os.MkdirAll(filepath.Join(append([]string{root}, elems...)...), 0o700)
	assert.NilError(t, err)
}

func writeFile(t *testing.T, root, name string) {
	t.Helper()

	err := os.WriteFile(filepath.Join(root, name), []byte("contents"), 0o600)
	assert.NilError(t, err)
}
