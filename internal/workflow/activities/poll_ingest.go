package activities

import (
	"context"
	"errors"
	"time"

	"github.com/artefactual-labs/enduro/internal/pipeline"
	wferrors "github.com/artefactual-labs/enduro/internal/workflow/errors"
	"github.com/artefactual-labs/enduro/internal/workflow/manager"
	"github.com/cenkalti/backoff/v3"
	"go.uber.org/cadence/activity"
)

type PollIngestActivity struct {
	manager *manager.Manager
}

func NewPollIngestActivity(m *manager.Manager) *PollIngestActivity {
	return &PollIngestActivity{manager: m}
}

type PollIngestActivityParams struct {
	PipelineName string
	SIPID        string
}

func (a *PollIngestActivity) Execute(ctx context.Context, params *PollIngestActivityParams) (time.Time, error) {
	p, err := a.manager.Pipelines.ByName(params.PipelineName)
	if err != nil {
		return time.Time{}, wferrors.NonRetryableError(err)
	}
	amc := p.Client()

	var backoffStrategy = backoff.WithContext(backoff.NewConstantBackOff(time.Second*5), ctx)

	err = backoff.RetryNotify(
		func() (err error) {
			ctx, cancel := context.WithTimeout(ctx, time.Second*2)
			defer cancel()

			err = pipeline.IngestStatus(ctx, amc, params.SIPID)
			if errors.Is(err, pipeline.ErrStatusNonRetryable) {
				return backoff.Permanent(wferrors.NonRetryableError(err))
			}

			return err
		},
		backoffStrategy,
		func(err error, duration time.Duration) {
			activity.RecordHeartbeat(ctx, err.Error())
		},
	)

	return time.Now().UTC(), err
}
