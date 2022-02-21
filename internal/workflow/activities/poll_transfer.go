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

// PollTransferActivity polls the Transfer Status API repeatedly until
// processing completes or when an error is considered non permanent.
//
// It is expected to deliver at least on heartbeat per minute.
type PollTransferActivity struct {
	manager *manager.Manager
}

func NewPollTransferActivity(m *manager.Manager) *PollTransferActivity {
	return &PollTransferActivity{manager: m}
}

type PollTransferActivityParams struct {
	PipelineName string
	TransferID   string
}

func (a *PollTransferActivity) Execute(ctx context.Context, params *PollTransferActivityParams) (string, error) {
	p, err := a.manager.Pipelines.ByName(params.PipelineName)
	if err != nil {
		return "", wferrors.NonRetryableError(err)
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

	return sipID, err
}
