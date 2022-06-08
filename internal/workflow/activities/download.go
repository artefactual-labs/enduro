package activities

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/artefactual-labs/enduro/internal/temporal"
	"github.com/artefactual-labs/enduro/internal/watcher"
)

// DownloadActivity downloads the blob into the processing directory.
type DownloadActivity struct {
	wsvc watcher.Service
}

func NewDownloadActivity(wsvc watcher.Service) *DownloadActivity {
	return &DownloadActivity{
		wsvc: wsvc,
	}
}

func tempFile(pattern string) (*os.File, error) {
	// XXX: inline this below?
	if pattern == "" {
		pattern = "blob-*"
	}
	return ioutil.TempFile("", pattern)
}

func (a *DownloadActivity) Execute(ctx context.Context, watcherName, key string) (string, error) {
	file, err := tempFile("blob-*")
	if err != nil {
		return "", temporal.NonRetryableError(fmt.Errorf("error creating temporary file in processing directory: %v", err))
	}
	defer file.Close()

	if err := a.wsvc.Download(ctx, file, watcherName, key); err != nil {
		return "", temporal.NonRetryableError(fmt.Errorf("error downloading blob: %v", err))
	}

	return file.Name(), nil
}
