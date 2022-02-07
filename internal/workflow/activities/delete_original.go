package activities

import (
	"context"
	"os"
	"path/filepath"

	"github.com/artefactual-labs/enduro/internal/workflow/manager"
)

type DeleteOriginalActivity struct {
	manager *manager.Manager
}

func NewDeleteOriginalActivity(m *manager.Manager) *DeleteOriginalActivity {
	return &DeleteOriginalActivity{manager: m}
}

func (a *DeleteOriginalActivity) Execute(ctx context.Context, watcherName, batchDir, key string) error {
	if batchDir != "" {
		return deleteOriginalFromBatch(batchDir, key)
	}
	return a.manager.Watcher.Delete(ctx, watcherName, key)
}

func deleteOriginalFromBatch(batchDir, key string) error {
	return os.RemoveAll(filepath.Join(batchDir, key))
}
