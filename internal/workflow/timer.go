package workflow

import (
	"sync"
	"time"

	temporalsdk_temporal "go.temporal.io/sdk/temporal"
	temporalsdk_workflow "go.temporal.io/sdk/workflow"
)

type Timer struct {
	exceeded bool
	sync.RWMutex
}

func NewTimer() *Timer {
	return &Timer{}
}

func (t *Timer) WithTimeout(ctx temporalsdk_workflow.Context, d time.Duration) (temporalsdk_workflow.Context, temporalsdk_workflow.CancelFunc) {
	logger := temporalsdk_workflow.GetLogger(ctx)

	timedCtx, cancelHandler := temporalsdk_workflow.WithCancel(ctx)
	temporalsdk_workflow.Go(ctx, func(ctx temporalsdk_workflow.Context) {
		if err := temporalsdk_workflow.NewTimer(ctx, d).Get(ctx, nil); err != nil {
			if !temporalsdk_temporal.IsCanceledError(err) {
				logger.Warn("Timer failed", "err", err.Error())
			}
		}

		cancelHandler()

		t.Lock()
		t.exceeded = true
		t.Unlock()
	})

	return timedCtx, cancelHandler
}

func (t *Timer) Exceeded() bool {
	t.RLock()
	defer t.RUnlock()
	return t.exceeded
}
