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

	timedCtx, cancelTimedCtx := temporalsdk_workflow.WithCancel(ctx)
	timerCtx, cancelTimer := temporalsdk_workflow.WithCancel(ctx)
	temporalsdk_workflow.Go(timerCtx, func(ctx temporalsdk_workflow.Context) {
		err := temporalsdk_workflow.NewTimer(ctx, d).Get(ctx, nil)
		if err != nil {
			if !temporalsdk_temporal.IsCanceledError(err) {
				logger.Warn("Timer failed", "err", err.Error())
			}
			return
		}

		t.Lock()
		t.exceeded = true
		t.Unlock()

		cancelTimedCtx()
	})

	return timedCtx, func() {
		cancelTimedCtx()
		cancelTimer()
	}
}

func (t *Timer) Exceeded() bool {
	t.RLock()
	defer t.RUnlock()
	return t.exceeded
}
