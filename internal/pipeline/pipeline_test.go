package pipeline

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestPipelineSemaphore(t *testing.T) {
	t.Parallel()

	p, err := NewPipeline(Config{Capacity: 3})
	assert.ErrorContains(t, err, "error during pipeline identification")

	tries := []bool{}

	// These three should succeed right away.
	tries = append(tries, p.TryAcquire())
	tries = append(tries, p.TryAcquire())
	tries = append(tries, p.TryAcquire())

	// And the one too because we've released once.
	p.Release()
	tries = append(tries, p.TryAcquire())

	// But this will fail because all the slots are taken.
	tries = append(tries, p.TryAcquire())

	assert.DeepEqual(t, tries, []bool{true, true, true, true, false})
}
