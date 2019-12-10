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

type PollTransferActivityParams struct {
	PipelineName string
	TransferID   string
}

func (a *PollTransferActivity) Execute(ctx context.Context, params *PollTransferActivityParams) (string, error) {
	p, err := a.manager.Pipelines.ByName(params.PipelineName)
	if err != nil {
		return "", err
	}
	amc := p.Client()

	var sipID string
	var backoffStrategy = backoff.WithContext(backoff.NewConstantBackOff(time.Second*5), ctx)

	err = backoff.RetryNotify(
		func() (err error) {
			ctx, cancel := context.WithTimeout(ctx, time.Second*2)
			defer cancel()

			sipID, err = pipeline.TransferStatus(ctx, amc, params.TransferID)
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
