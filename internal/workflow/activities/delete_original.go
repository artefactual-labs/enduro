package activities

import (
	"context"
	"os"
	"path/filepath"

	"github.com/artefactual-labs/enduro/internal/watcher"
)

type DeleteOriginalActivity struct {
	wsvc watcher.Service
}

func NewDeleteOriginalActivity(wsvc watcher.Service) *DeleteOriginalActivity {
	return &DeleteOriginalActivity{wsvc: wsvc}
}

func (a *DeleteOriginalActivity) Execute(ctx context.Context, watcherName, batchDir, key string) error {
	if batchDir != "" {
		return deleteOriginalFromBatch(batchDir, key)
	}
	return a.wsvc.Delete(ctx, watcherName, key)
}

func deleteOriginalFromBatch(batchDir, key string) error {
	return os.RemoveAll(filepath.Join(batchDir, key))
}
