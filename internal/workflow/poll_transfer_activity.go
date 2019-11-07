package workflow

import (
	"context"
	"errors"
	"time"

	"github.com/artefactual-labs/enduro/internal/pipeline"
	"github.com/cenkalti/backoff/v3"
	"go.uber.org/cadence/activity"
)

type PollTransferActivity struct {
	manager *Manager
}

func NewPollTransferActivity(m *Manager) *PollTransferActivity {
	return &PollTransferActivity{manager: m}
}

func (a *PollTransferActivity) Execute(ctx context.Context, tinfo *TransferInfo) (string, error) {
	amc, err := a.manager.Pipelines.Client(tinfo.Event.PipelineName)
	if err != nil {
		return "", err
	}

	var sipID string
	var backoffStrategy = backoff.WithContext(backoff.NewConstantBackOff(time.Second*5), ctx)

	err = backoff.RetryNotify(
		func() (err error) {
			ctx, cancel := context.WithTimeout(ctx, time.Second*2)
			defer cancel()

			sipID, err = pipeline.TransferStatus(ctx, amc, tinfo.TransferID)
			if errors.Is(err, pipeline.ErrStatusNonRetryable) {
				return backoff.Permanent(nonRetryableError(err))
			}

			return err
		},
		backoffStrategy,
		func(err error, duration time.Duration) {
			activity.RecordHeartbeat(ctx, err.Error())
		},
	)

	return sipID, err
}
