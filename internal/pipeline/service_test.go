package pipeline

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-logr/logr"
	"gotest.tools/v3/assert"

	goapipeline "github.com/artefactual-labs/enduro/internal/api/gen/pipeline"
)

func TestService(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	logger := logr.Discard()

	ams := amserver(t)

	registry, _ := NewPipelineRegistry(
		logger,
		[]Config{
			{
				Name:    "am1",
				ID:      "0e395063-b859-45a3-8999-8f4116bb62e9",
				BaseURL: ams.URL,
			},
			{
				Name: "am2",
			},
		},
	)

	svc := NewService(logger, registry)

	listresp, err := svc.List(ctx, &goapipeline.ListPayload{})
	assert.NilError(t, err)
	assert.Equal(t, len(listresp), 2)

	showresp, err := svc.Show(ctx, &goapipeline.ShowPayload{ID: "0e395063-b859-45a3-8999-8f4116bb62e9"})
	assert.NilError(t, err)
	assert.Equal(t, showresp.Name, "am1")

	showresp, err = svc.Show(ctx, &goapipeline.ShowPayload{ID: "12345"})
	assert.Error(t, err, "unknown pipeline")
	assert.Assert(t, showresp == nil)

	processingresp, err := svc.Processing(ctx, &goapipeline.ProcessingPayload{ID: "12345"})
	assert.Error(t, err, "Pipeline not found.")
	assert.Assert(t, processingresp == nil)

	registry.pipelines["am1"].client = ams.Client()
	processingresp, err = svc.Processing(ctx, &goapipeline.ProcessingPayload{ID: "0e395063-b859-45a3-8999-8f4116bb62e9"})
	assert.NilError(t, err)
	assert.DeepEqual(t, processingresp, []string{"automated", "default"})
}

func amserver(t *testing.T) *httptest.Server {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `{"processing_configurations": ["automated", "default"]}`)
	}))
	defer t.Cleanup(func() {
		ts.Close()
	})
	return ts
}
