package activities

import (
	"context"

	"github.com/artefactual-labs/enduro/internal/watcher"
	"github.com/artefactual-labs/enduro/internal/workflow/manager"
)

type DeleteOriginalActivity struct {
	manager *manager.Manager
}

func NewDeleteOriginalActivity(m *manager.Manager) *DeleteOriginalActivity {
	return &DeleteOriginalActivity{manager: m}
}

func (a *DeleteOriginalActivity) Execute(ctx context.Context, event *watcher.BlobEvent) error {
	return a.manager.Watcher.Delete(ctx, event)
}
