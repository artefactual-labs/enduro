package activities

import (
	"context"
	"fmt"

	"github.com/artefactual-labs/enduro/internal/pipeline"
	"github.com/artefactual-labs/enduro/internal/temporal"
	"github.com/artefactual-labs/enduro/internal/watcher"
	"github.com/artefactual-labs/enduro/internal/workflow/hooks"
)

// DownloadActivity downloads the blob into the pipeline processing directory.
type DownloadActivity struct {
	hooks            *hooks.Hooks
	wsvc             watcher.Service
	pipelineRegistry *pipeline.Registry
}

func NewDownloadActivity(h *hooks.Hooks, pipelineRegistry *pipeline.Registry, wsvc watcher.Service) *DownloadActivity {
	return &DownloadActivity{hooks: h, pipelineRegistry: pipelineRegistry, wsvc: wsvc}
}

func (a *DownloadActivity) Execute(ctx context.Context, pipelineName, watcherName, key string) (string, error) {
	p, err := a.pipelineRegistry.ByName(pipelineName)
	if err != nil {
		return "", temporal.NewNonRetryableError(err)
	}

	file, err := p.TempFile("blob-*")
	if err != nil {
		return "", temporal.NewNonRetryableError(fmt.Errorf("error creating temporary file in processing directory: %v", err))
	}
	defer file.Close()

	if err := a.wsvc.Download(ctx, file, watcherName, key); err != nil {
		return "", temporal.NewNonRetryableError(fmt.Errorf("error downloading blob: %v", err))
	}

	return file.Name(), nil
}
