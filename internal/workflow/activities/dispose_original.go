package activities

import (
	"context"
	"path/filepath"

	"github.com/artefactual-labs/enduro/internal/fsutil"
	"github.com/artefactual-labs/enduro/internal/workflow/manager"
)

type DisposeOriginalActivity struct {
	manager *manager.Manager
}

func NewDisposeOriginalActivity(m *manager.Manager) *DisposeOriginalActivity {
	return &DisposeOriginalActivity{manager: m}
}

func (a *DisposeOriginalActivity) Execute(ctx context.Context, watcherName, completedDir, batchDir, key string) error {
	if batchDir != "" {
		return disposeOriginalFromBatch(completedDir, batchDir, key)
	}
	return a.manager.Watcher.Dispose(ctx, watcherName, key)
}

func disposeOriginalFromBatch(completedDir, batchDir, key string) error {
	if completedDir == "" {
		return nil
	}

	src := filepath.Join(batchDir, key)
	dst := filepath.Join(completedDir, key)

	return fsutil.Move(src, dst)
}
