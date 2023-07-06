package activities

import (
	"context"
	"errors"
	"time"

	"github.com/cenkalti/backoff/v4"
	temporalsdk_activity "go.temporal.io/sdk/activity"

	"github.com/artefactual-labs/enduro/internal/pipeline"
	"github.com/artefactual-labs/enduro/internal/temporal"
)

type PollIngestActivity struct {
	pipelineRegistry *pipeline.Registry
}

func NewPollIngestActivity(pipelineRegistry *pipeline.Registry) *PollIngestActivity {
	return &PollIngestActivity{pipelineRegistry: pipelineRegistry}
}

type PollIngestActivityParams struct {
	PipelineName string
	SIPID        string
}

func (a *PollIngestActivity) Execute(ctx context.Context, params *PollIngestActivityParams) (time.Time, error) {
	logger := temporalsdk_activity.GetLogger(ctx)

	p, err := a.pipelineRegistry.ByName(params.PipelineName)
	if err != nil {
		return time.Time{}, temporal.NewNonRetryableError(err)
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
				return backoff.Permanent(temporal.NewNonRetryableError(err))
			}

			// Looking good, keep polling.
			if errors.Is(err, pipeline.ErrStatusInProgress) {
				lastRetryableError = time.Time{} // Reset.
				return err
			}

			if err != nil {
				logger.Error("Failed to look up Ingest status.", "error", err)
			}

			// Retry unless the deadline was exceeded.
			if lastRetryableError.IsZero() {
				lastRetryableError = clock.Now()
			} else if clock.Since(lastRetryableError) > deadline {
				return backoff.Permanent(temporal.NewNonRetryableError(err))
			}

			return err
		},
		backoffStrategy,
		func(err error, duration time.Duration) {
			temporalsdk_activity.RecordHeartbeat(ctx, err.Error())
		},
	)

	return clock.Now().UTC(), err
}
