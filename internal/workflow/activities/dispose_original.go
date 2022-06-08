package activities

import (
	"context"
	"path/filepath"

	"github.com/artefactual-labs/enduro/internal/fsutil"
	"github.com/artefactual-labs/enduro/internal/watcher"
)

type DisposeOriginalActivity struct {
	wsvc watcher.Service
}

func NewDisposeOriginalActivity(wsvc watcher.Service) *DisposeOriginalActivity {
	return &DisposeOriginalActivity{wsvc: wsvc}
}

func (a *DisposeOriginalActivity) Execute(ctx context.Context, watcherName, completedDir, batchDir, key string) error {
	if batchDir != "" {
		return disposeOriginalFromBatch(completedDir, batchDir, key)
	}
	return a.wsvc.Dispose(ctx, watcherName, key)
}

func disposeOriginalFromBatch(completedDir, batchDir, key string) error {
	if completedDir == "" {
		return nil
	}

	src := filepath.Join(batchDir, key)
	dst := filepath.Join(completedDir, key)

	return fsutil.Move(src, dst)
}
