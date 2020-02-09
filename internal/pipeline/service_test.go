package pipeline

import (
	"context"
	"testing"

	goapipeline "github.com/artefactual-labs/enduro/internal/api/gen/pipeline"
	logrtesting "github.com/go-logr/logr/testing"
	"gotest.tools/v3/assert"
)

func TestService(t *testing.T) {
	ctx := context.Background()
	logger := logrtesting.NullLogger{}

	registry, _ := NewPipelineRegistry(
		logger,
		[]Config{
			{
				Name: "am1",
				ID:   "0e395063-b859-45a3-8999-8f4116bb62e9",
			},
			{
				Name: "am2",
			},
		},
	)

	svc := NewService(logger, registry)

	var err error

	listresp, err := svc.List(ctx, &goapipeline.ListPayload{})
	assert.NilError(t, err)
	assert.Equal(t, len(listresp), 2)

	showresp, err := svc.Show(ctx, &goapipeline.ShowPayload{ID: "0e395063-b859-45a3-8999-8f4116bb62e9"})
	assert.NilError(t, err)
	assert.Equal(t, showresp.Name, "am1")

	showresp, err = svc.Show(ctx, &goapipeline.ShowPayload{ID: "12345"})
	assert.Error(t, err, "unknown pipeline")
	assert.Assert(t, showresp == nil)
}
