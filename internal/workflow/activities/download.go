package activities

import (
	"context"
	"fmt"

	wferrors "github.com/artefactual-labs/enduro/internal/workflow/errors"
	"github.com/artefactual-labs/enduro/internal/workflow/manager"
)

// DownloadActivity downloads the blob into the pipeline processing directory.
type DownloadActivity struct {
	manager *manager.Manager
}

func NewDownloadActivity(m *manager.Manager) *DownloadActivity {
	return &DownloadActivity{manager: m}
}

func (a *DownloadActivity) Execute(ctx context.Context, pipelineName, watcherName, key string) (string, error) {
	p, err := a.manager.Pipelines.ByName(pipelineName)
	if err != nil {
		return "", wferrors.NonRetryableError(err)
	}

	file, err := p.TempFile("blob-*")
	if err != nil {
		return "", wferrors.NonRetryableError(fmt.Errorf("error creating temporary file in processing directory: %v", err))
	}
	defer file.Close()

	if err := a.manager.Watcher.Download(ctx, file, watcherName, key); err != nil {
		return "", wferrors.NonRetryableError(fmt.Errorf("error downloading blob: %v", err))
	}

	return file.Name(), nil
}
