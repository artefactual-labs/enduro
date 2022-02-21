package activities

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"gotest.tools/v3/assert"

	"github.com/artefactual-labs/enduro/internal/pipeline"
	"github.com/artefactual-labs/enduro/internal/workflow/manager"
)

type ZeroBackOff struct {
	hits int
}

func (b *ZeroBackOff) Reset() {}

func (b *ZeroBackOff) NextBackOff() time.Duration {
	b.hits++
	return 0
}

func newManager(t *testing.T, h http.HandlerFunc) *manager.Manager {
	t.Helper()
	logger := logr.Discard()

	mux := http.NewServeMux()
	if h != nil {
		mux.HandleFunc("/", h)
	}

	server := httptest.NewServer(mux)
	t.Cleanup(func() { server.Close() })

	retryDeadline := time.Duration(time.Minute * 5)
	pipelineURL, _ := url.Parse(server.URL)
	pipelines, err := pipeline.NewPipelineRegistry(logger, []pipeline.Config{
		{
			ID:            "75ed5f0a-792e-4ce9-aeb7-e2e832d2a4fa",
			Name:          "am",
			BaseURL:       pipelineURL.String(),
			RetryDeadline: &retryDeadline,
		},
	})
	assert.NilError(t, err)

	return &manager.Manager{
		Logger:    logger,
		Pipelines: pipelines,
	}
}
