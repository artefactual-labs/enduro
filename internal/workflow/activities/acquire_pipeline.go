package activities

import (
	"context"
	"time"

	"github.com/artefactual-labs/enduro/internal/workflow/manager"
)

// AcquirePipelineActivity acquires a lock in the weighted semaphore associated
// to a particular pipeline.
type AcquirePipelineActivity struct {
	manager *manager.Manager
}

func NewAcquirePipelineActivity(m *manager.Manager) *AcquirePipelineActivity {
	return &AcquirePipelineActivity{manager: m}
}

func (a *AcquirePipelineActivity) Execute(ctx context.Context, name string) error {
	p, err := a.manager.Pipelines.ByName(name)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	return p.Acquire(ctx)
}
