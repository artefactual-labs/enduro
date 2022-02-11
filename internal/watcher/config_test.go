package watcher_test

import (
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/artefactual-labs/enduro/internal/watcher"
)

func TestCompletedDirs(t *testing.T) {
	c := watcher.Config{
		Filesystem: []*watcher.FilesystemConfig{
			{CompletedDir: ""},
			nil,
			{CompletedDir: "/tmp/test-1"},
			{CompletedDir: "/tmp/test-2"},
			{CompletedDir: "./test-3"},
		},
	}

	wd, _ := os.Getwd()
	assert.DeepEqual(t, c.CompletedDirs(), []string{
		"/tmp/test-1",
		"/tmp/test-2",
		filepath.Join(wd, "test-3"),
	})
}
