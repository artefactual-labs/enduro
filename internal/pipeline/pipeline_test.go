package pipeline

import (
	"testing"

	"github.com/go-logr/logr"
	"gotest.tools/v3/assert"
)

func TestPipelineSemaphore(t *testing.T) {
	t.Parallel()

	p, err := NewPipeline(logr.Discard(), Config{Capacity: 3})
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

	t.Run("Release panics are gracefully managed", func(t *testing.T) {
		t.Parallel()

		p, _ := NewPipeline(logr.Discard(), Config{Capacity: 3})

		defer func() {
			err := recover()
			assert.Equal(t, err, nil)
		}()

		for i := 0; i < 10; i++ {
			p.Release()
		}
	})

	t.Run("Weight cannot go below zero", func(t *testing.T) {
		t.Parallel()

		p, _ := NewPipeline(logr.Discard(), Config{Capacity: 3})

		for i := 0; i < 50; i++ {
			p.Release()
		}

		tries := []bool{}
		tries = append(tries, p.TryAcquire())
		tries = append(tries, p.TryAcquire())
		tries = append(tries, p.TryAcquire())
		tries = append(tries, p.TryAcquire())

		assert.DeepEqual(t, tries, []bool{true, true, true, false})
	})
}
