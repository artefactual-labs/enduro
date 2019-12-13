package activities

import (
	"context"
	"errors"
	"fmt"

	"github.com/artefactual-labs/enduro/internal/watcher"
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

func (a *DownloadActivity) Execute(ctx context.Context, event *watcher.BlobEvent) (string, error) {
	if event == nil {
		return "", wferrors.NonRetryableError(errors.New("error reading parameters"))
	}
	p, err := a.manager.Pipelines.ByName(event.PipelineName)
	if err != nil {
		return "", wferrors.NonRetryableError(err)
	}

	file, err := p.TempFile("blob-*")
	if err != nil {
		return "", wferrors.NonRetryableError(fmt.Errorf("error creating temporary file in processing directory: %v", err))
	}
	defer file.Close()

	if err := a.manager.Watcher.Download(ctx, file, event); err != nil {
		return "", wferrors.NonRetryableError(fmt.Errorf("error downloading blob: %v", err))
	}

	return file.Name(), nil
}
