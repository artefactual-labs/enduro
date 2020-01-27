package activities

import (
	"context"
	"time"

	wferrors "github.com/artefactual-labs/enduro/internal/workflow/errors"
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

func (a *AcquirePipelineActivity) Execute(ctx context.Context, pipelineName string) error {
	p, err := a.manager.Pipelines.ByName(pipelineName)
	if err != nil {
		return wferrors.NonRetryableError(err)
	}

	ctx, cancel := context.WithTimeout(ctx, time.Second*4)
	defer cancel()

	return p.Acquire(ctx)
}
