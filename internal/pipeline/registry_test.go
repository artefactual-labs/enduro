package pipeline_test

import (
	"sort"
	"testing"

	"github.com/go-logr/logr"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"

	"github.com/artefactual-labs/enduro/internal/pipeline"
)

func TestRegistryByName(t *testing.T) {
	config := []pipeline.Config{
		{Name: "am1"},
		{Name: "am2"},
	}

	registry, err := pipeline.NewPipelineRegistry(logr.Discard(), config, nil, nil)
	assert.NilError(t, err)

	pipe, err := registry.ByName("am1")
	assert.NilError(t, err)
	assert.Equal(t, pipe.Config().Name, "am1")

	pipe, err = registry.ByName("am2")
	assert.NilError(t, err)
	assert.Equal(t, pipe.Config().Name, "am2")

	pipe, err = registry.ByName("am3")
	assert.ErrorIs(t, err, pipeline.ErrUnknownPipeline)
	assert.Assert(t, is.Nil(pipe))
}

func TestRegistryNames(t *testing.T) {
	config := []pipeline.Config{
		{Name: "am1"},
		{Name: "am2"},
	}

	registry, err := pipeline.NewPipelineRegistry(logr.Discard(), config, nil, nil)
	names := registry.Names()

	assert.NilError(t, err)
	sort.Strings(names)
	assert.DeepEqual(t, names, []string{"am1", "am2"})
}

func TestRegistryRejectsInvalidRecoveryConfig(t *testing.T) {
	t.Parallel()

	_, err := pipeline.NewPipelineRegistry(logr.Discard(), []pipeline.Config{
		{
			Name: "am1",
			Recovery: pipeline.RecoveryConfig{
				ReconcileExistingAIP: true,
			},
		},
	}, nil, nil)

	assert.ErrorContains(t, err, `pipeline "am1": invalid recovery configuration`)
	assert.ErrorContains(t, err, "storageServiceURL is required")
}

func TestRegistrySkipsPipelinesThatFailConstruction(t *testing.T) {
	t.Parallel()

	registry, err := pipeline.NewPipelineRegistry(logr.Discard(), []pipeline.Config{
		{
			Name:              "healthy",
			ID:                "healthy-id",
			StorageServiceURL: "https://storage-service.example.com",
		},
		{
			Name:              "broken",
			StorageServiceURL: "://bad-url",
		},
	}, nil, nil)
	assert.NilError(t, err)

	pipe, err := registry.ByName("healthy")
	assert.NilError(t, err)
	assert.Equal(t, pipe.Config().Name, "healthy")

	pipe, err = registry.ByName("broken")
	assert.ErrorIs(t, err, pipeline.ErrUnknownPipeline)
	assert.Assert(t, is.Nil(pipe))

	names := registry.Names()
	sort.Strings(names)
	assert.DeepEqual(t, names, []string{"healthy"})
}
