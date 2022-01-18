package activities

import (
	"context"

	"github.com/artefactual-labs/enduro/internal/workflow/manager"
)

type DisposeOriginalActivity struct {
	manager *manager.Manager
}

func NewDisposeOriginalActivity(m *manager.Manager) *DisposeOriginalActivity {
	return &DisposeOriginalActivity{manager: m}
}

func (a *DisposeOriginalActivity) Execute(ctx context.Context, watcherName, key string) error {
	return a.manager.Watcher.Dispose(ctx, watcherName, key)
}
