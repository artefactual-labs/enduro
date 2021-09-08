package workflow

import (
	"sync"
	"time"

	"go.uber.org/cadence/workflow"
	"go.uber.org/zap"
)

type Timer struct {
	exceeded bool
	sync.RWMutex
}

func NewTimer() *Timer {
	return &Timer{}
}

func (t *Timer) WithTimeout(ctx workflow.Context, d time.Duration) (workflow.Context, workflow.CancelFunc) {
	logger := workflow.GetLogger(ctx)

	timedCtx, cancelHandler := workflow.WithCancel(ctx)
	workflow.Go(ctx, func(ctx workflow.Context) {
		if err := workflow.NewTimer(ctx, d).Get(ctx, nil); err != nil {
			logger.Warn("Timer failed", zap.Error(err))
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
