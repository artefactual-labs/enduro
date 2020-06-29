package activities

import (
	"context"

	"github.com/artefactual-labs/enduro/internal/workflow/manager"
)

type DeleteOriginalActivity struct {
	manager *manager.Manager
}

func NewDeleteOriginalActivity(m *manager.Manager) *DeleteOriginalActivity {
	return &DeleteOriginalActivity{manager: m}
}

func (a *DeleteOriginalActivity) Execute(ctx context.Context, watcherName, key string) error {
	return a.manager.Watcher.Delete(ctx, watcherName, key)
}
