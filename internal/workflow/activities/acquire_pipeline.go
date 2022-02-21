package activities

import (
	"context"
	"fmt"
	"time"

	"github.com/cenkalti/backoff/v4"
	cadencesdk_activity "go.uber.org/cadence/activity"

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

	errAcquirePipeline := fmt.Errorf("error acquring semaphore: busy")

	err = backoff.RetryNotify(
		func() (err error) {
			ok := p.TryAcquire()
			if !ok {
				err = errAcquirePipeline
			}

			return err
		},
		backoff.WithContext(backoff.NewConstantBackOff(time.Second*5), ctx),
		func(err error, duration time.Duration) {
			cadencesdk_activity.RecordHeartbeat(ctx)
		},
	)

	return err
}
