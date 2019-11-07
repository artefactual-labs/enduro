package workflow

import (
	"context"

	"github.com/artefactual-labs/enduro/internal/watcher"
)

type DeleteOriginalActivity struct {
	manager *Manager
}

func NewDeleteOriginalActivity(m *Manager) *DeleteOriginalActivity {
	return &DeleteOriginalActivity{manager: m}
}

func (a *DeleteOriginalActivity) Execute(ctx context.Context, event *watcher.BlobEvent) error {
	return a.manager.Watcher.Delete(ctx, event)
}
