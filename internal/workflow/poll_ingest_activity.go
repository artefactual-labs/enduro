package workflow

import (
	"context"
	"errors"
	"time"

	"github.com/artefactual-labs/enduro/internal/pipeline"
	"github.com/cenkalti/backoff/v3"
	"go.uber.org/cadence/activity"
)

type PollIngestActivity struct {
	manager *Manager
}

func NewPollIngestActivity(m *Manager) *PollIngestActivity {
	return &PollIngestActivity{manager: m}
}

func (a *PollIngestActivity) Execute(ctx context.Context, tinfo *TransferInfo) (time.Time, error) {
	amc, err := a.manager.Pipelines.Client(tinfo.Event.PipelineName)
	if err != nil {
		return time.Time{}, err
	}

	var backoffStrategy = backoff.WithContext(backoff.NewConstantBackOff(time.Second*5), ctx)

	err = backoff.RetryNotify(
		func() (err error) {
			ctx, cancel := context.WithTimeout(ctx, time.Second*2)
			defer cancel()

			err = pipeline.IngestStatus(ctx, amc, tinfo.SIPID)
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

	return time.Now().UTC(), err
}
