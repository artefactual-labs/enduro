package workflow

import (
	"context"
	"errors"
	"fmt"

	"github.com/artefactual-labs/enduro/internal/watcher"
)

// DownloadActivity downloads the blob into the pipeline processing directory.
type DownloadActivity struct {
	manager *Manager
}

func NewDownloadActivity(m *Manager) *DownloadActivity {
	return &DownloadActivity{manager: m}
}

func (a *DownloadActivity) Execute(ctx context.Context, event *watcher.BlobEvent) (string, error) {
	if event == nil {
		return "", nonRetryableError(errors.New("error reading parameters"))
	}
	p, err := a.manager.Pipelines.Pipeline(event.PipelineName)
	if err != nil {
		return "", nonRetryableError(err)
	}

	file, err := p.TempFile("blob-*")
	if err != nil {
		return "", nonRetryableError(fmt.Errorf("error creating temporary file in processing directory: %v", err))
	}
	defer file.Close()

	if err := a.manager.Watcher.Download(ctx, file, event); err != nil {
		return "", nonRetryableError(fmt.Errorf("error downloading blob: %v", err))
	}

	return file.Name(), nil
}
