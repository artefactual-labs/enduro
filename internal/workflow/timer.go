package workflow

import (
	"sync"
	"time"

	cadencesdk "go.uber.org/cadence"
	cadencesdk_workflow "go.uber.org/cadence/workflow"
	"go.uber.org/zap"
)

type Timer struct {
	exceeded bool
	sync.RWMutex
}

func NewTimer() *Timer {
	return &Timer{}
}

func (t *Timer) WithTimeout(ctx cadencesdk_workflow.Context, d time.Duration) (cadencesdk_workflow.Context, cadencesdk_workflow.CancelFunc) {
	logger := cadencesdk_workflow.GetLogger(ctx)

	timedCtx, cancelHandler := cadencesdk_workflow.WithCancel(ctx)
	cadencesdk_workflow.Go(ctx, func(ctx cadencesdk_workflow.Context) {
		if err := cadencesdk_workflow.NewTimer(ctx, d).Get(ctx, nil); err != nil {
			if !cadencesdk.IsCanceledError(err) {
				logger.Warn("Timer failed", zap.Error(err))
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
