package workflow

import (
	"context"
	"fmt"
	"net/http"

	"github.com/artefactual-labs/enduro/internal/amclient"
)

// TransferActivity submits the transfer to Archivematica and returns its ID.
//
// This is our first interaction with Archivematica. The workflow ends here
// after authentication errors.
type TransferActivity struct {
	manager *Manager
}

func NewTransferActivity(m *Manager) *TransferActivity {
	return &TransferActivity{manager: m}
}

func (a *TransferActivity) Execute(ctx context.Context, tinfo *TransferInfo) (string, error) {
	amc, err := a.manager.Pipelines.Client(tinfo.PipelineConfig.Name)
	if err != nil {
		return "", err
	}

	// Transfer path should include the location UUID if defined.
	var path = tinfo.Bundle.RelPath
	if tinfo.PipelineConfig.TransferLocationID != "" {
		path = fmt.Sprintf("%s:%s", tinfo.PipelineConfig.TransferLocationID, path)
	}

	var autoApprove = true
	resp, httpResp, err := amc.Package.Create(ctx, &amclient.PackageCreateRequest{
		Name:             tinfo.Bundle.Name,
		Path:             path,
		ProcessingConfig: tinfo.PipelineConfig.ProcessingConfig,
		AutoApprove:      &autoApprove,
	})
	if err != nil {
		if httpResp != nil {
			switch {
			case httpResp.StatusCode == http.StatusForbidden:
				return "", nonRetryableError(fmt.Errorf("authentication error: %v", err))
			}
		}
		return "", err
	}

	return resp.ID, nil
}
