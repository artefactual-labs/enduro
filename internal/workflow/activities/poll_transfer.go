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

// PollTransferActivity polls the Transfer Status API repeatedly until
// processing completes or when an error is considered non permanent.
//
// It is expected to deliver at least on heartbeat per minute.
type PollTransferActivity struct {
	pipelineRegistry *pipeline.Registry
}

func NewPollTransferActivity(pipelineRegistry *pipeline.Registry) *PollTransferActivity {
	return &PollTransferActivity{pipelineRegistry: pipelineRegistry}
}

type PollTransferActivityParams struct {
	PipelineName string
	TransferID   string
}

func (a *PollTransferActivity) Execute(ctx context.Context, params *PollTransferActivityParams) (string, error) {
	logger := temporalsdk_activity.GetLogger(ctx)

	p, err := a.pipelineRegistry.ByName(params.PipelineName)
	if err != nil {
		return "", temporal.NewNonRetryableError(err)
	}
	amc := p.Client()

	deadline := defaultMaxElapsedTime
	if retryDeadline := p.Config().RetryDeadline; retryDeadline != nil {
		deadline = *retryDeadline
	}

	var sipID string
	lastRetryableError := time.Time{}
	backoffStrategy := backoff.WithContext(backoffStrategy, ctx)

	err = backoff.RetryNotify(
		func() (err error) {
			ctx, cancel := context.WithTimeout(ctx, time.Second*10)
			defer cancel()

			sipID, err = pipeline.TransferStatus(ctx, amc.Transfer, params.TransferID)

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
				logger.Error("Failed to look up Transfer status.", "error", err)
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

	return sipID, err
}
