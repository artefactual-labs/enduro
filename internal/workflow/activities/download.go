package activities

import (
	"context"
	"fmt"

	"github.com/artefactual-labs/enduro/internal/temporal"
	"github.com/artefactual-labs/enduro/internal/watcher"
	"github.com/artefactual-labs/enduro/internal/workflow/manager"
)

// DownloadActivity downloads the blob into the pipeline processing directory.
type DownloadActivity struct {
	manager *manager.Manager
	wsvc    watcher.Service
}

func NewDownloadActivity(m *manager.Manager, wsvc watcher.Service) *DownloadActivity {
	return &DownloadActivity{manager: m, wsvc: wsvc}
}

func (a *DownloadActivity) Execute(ctx context.Context, pipelineName, watcherName, key string) (string, error) {
	p, err := a.manager.Pipelines.ByName(pipelineName)
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
