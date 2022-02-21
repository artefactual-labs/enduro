package activities

import (
	"context"
	"errors"
	"time"

	"github.com/cenkalti/backoff/v4"
	cadencesdk_activity "go.uber.org/cadence/activity"

	"github.com/artefactual-labs/enduro/internal/pipeline"
	wferrors "github.com/artefactual-labs/enduro/internal/workflow/errors"
	"github.com/artefactual-labs/enduro/internal/workflow/manager"
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

	deadline := defaultMaxElapsedTime
	if retryDeadline := p.Config().RetryDeadline; retryDeadline != nil {
		deadline = *retryDeadline
	}

	backoffStrategy := backoff.WithContext(backoffStrategy, ctx)
	lastRetryableError := time.Time{}

	err = backoff.RetryNotify(
		func() (err error) {
			ctx, cancel := context.WithTimeout(ctx, time.Second*10)
			defer cancel()

			err = pipeline.IngestStatus(ctx, amc.Ingest, params.SIPID)

			// Abandon when we see a non-retryable error.
			if errors.Is(err, pipeline.ErrStatusNonRetryable) {
				return backoff.Permanent(wferrors.NonRetryableError(err))
			}

			// Looking good, keep polling.
			if errors.Is(err, pipeline.ErrStatusInProgress) {
				lastRetryableError = time.Time{} // Reset.
				return err
			}

			// Retry unless the deadline was exceeded.
			if lastRetryableError.IsZero() {
				lastRetryableError = clock.Now()
			} else if clock.Since(lastRetryableError) > deadline {
				return backoff.Permanent(wferrors.NonRetryableError(err))
			}

			return err
		},
		backoffStrategy,
		func(err error, duration time.Duration) {
			cadencesdk_activity.RecordHeartbeat(ctx, err.Error())
		},
	)

	return clock.Now().UTC(), err
}
