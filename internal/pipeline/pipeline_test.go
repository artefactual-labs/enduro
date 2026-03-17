package pipeline

import (
	"testing"

	"github.com/go-logr/logr"
	"gotest.tools/v3/assert"
)

func TestPipelineSemaphore(t *testing.T) {
	t.Parallel()

	p, err := NewPipeline(logr.Discard(), Config{Capacity: 3}, nil, nil)
	assert.ErrorContains(t, err, "error during pipeline identification")

	tries := []bool{}

	testCapacity(t, p, 3, 0)

	// These three should succeed right away.
	tries = append(tries, p.TryAcquire())
	testCapacity(t, p, 3, 1)
	tries = append(tries, p.TryAcquire())
	testCapacity(t, p, 3, 2)
	tries = append(tries, p.TryAcquire())
	testCapacity(t, p, 3, 3)

	// And the one too because we've released once.
	p.Release()
	testCapacity(t, p, 3, 2)
	tries = append(tries, p.TryAcquire())

	// But this will fail because all the slots are taken.
	tries = append(tries, p.TryAcquire())

	assert.DeepEqual(t, tries, []bool{true, true, true, true, false})
	testCapacity(t, p, 3, 3)

	t.Run("Release panics are gracefully managed", func(t *testing.T) {
		t.Parallel()

		p, _ := NewPipeline(logr.Discard(), Config{Capacity: 3}, nil, nil)

		defer func() {
			err := recover()
			assert.Equal(t, err, nil)
		}()

		for range 10 {
			p.Release()
		}
	})

	t.Run("Weight cannot go below zero", func(t *testing.T) {
		t.Parallel()

		p, _ := NewPipeline(logr.Discard(), Config{Capacity: 3}, nil, nil)

		for range 50 {
			p.Release()
		}

		tries := []bool{}
		tries = append(tries, p.TryAcquire())
		tries = append(tries, p.TryAcquire())
		tries = append(tries, p.TryAcquire())
		tries = append(tries, p.TryAcquire())

		assert.DeepEqual(t, tries, []bool{true, true, true, false})
		testCapacity(t, p, 3, 3)
	})
}

func TestPipelineConfigValidate(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		cfg         Config
		errContains string
	}{
		"Defaults are safe": {
			cfg: Config{},
		},
		"Storage Service URL alone does not enable recovery": {
			cfg: Config{StorageServiceURL: "http://user:key@example.com"},
		},
		"Reconciliation requires Storage Service URL": {
			cfg: Config{
				Recovery: RecoveryConfig{
					ReconcileExistingAIP: true,
				},
			},
			errContains: "storageServiceURL is required",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := tc.cfg.Validate()
			if tc.errContains == "" {
				assert.NilError(t, err)
				return
			}

			assert.ErrorContains(t, err, tc.errContains)
		})
	}
}

func testCapacity(t *testing.T, p *Pipeline, s, c int64) {
	t.Helper()

	size, cur := p.Capacity()

	got := []int64{size, cur}
	want := []int64{s, c}

	assert.DeepEqual(t, got, want)
}
