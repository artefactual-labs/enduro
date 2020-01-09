package pipeline

import (
	"context"
	"testing"
	"time"

	"gotest.tools/v3/assert"
)

func TestPipelineSemaphore(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	p, err := NewPipeline(Config{Capacity: 3})
	assert.ErrorContains(t, err, "error during pipeline identification")

	tryAcquire := func(n int64) bool {
		ctx, cancel := context.WithTimeout(ctx, 10*time.Millisecond)
		defer cancel()
		return p.Acquire(ctx) == nil
	}

	tries := []bool{}

	// These three should succeed right away.
	tries = append(tries, tryAcquire(1))
	tries = append(tries, tryAcquire(1))
	tries = append(tries, tryAcquire(1))

	// And the one too because we've released once.
	p.Release()
	tries = append(tries, tryAcquire(1))

	// But this will fail because all the slots are taken.
	tries = append(tries, tryAcquire(1))

	assert.DeepEqual(t, tries, []bool{true, true, true, true, false})
}
