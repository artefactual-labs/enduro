package activities

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/artefactual-labs/enduro/internal/amclient"
	wferrors "github.com/artefactual-labs/enduro/internal/workflow/errors"
	"github.com/artefactual-labs/enduro/internal/workflow/manager"

	"github.com/cenkalti/backoff/v3"
	"go.uber.org/cadence/activity"
)

// TransferActivity submits the transfer to Archivematica and returns its ID.
//
// This is our first interaction with Archivematica. The workflow ends here
// after authentication errors.
type TransferActivity struct {
	manager *manager.Manager
}

func NewTransferActivity(m *manager.Manager) *TransferActivity {
	return &TransferActivity{manager: m}
}

type TransferActivityParams struct {
	PipelineName       string
	TransferLocationID string
	RelPath            string
	Name               string
	ProcessingConfig   string
	AutoApprove        bool
}

type TransferActivityResponse struct {
	TransferID      string
	PipelineVersion string
	PipelineID      string
}

func (a *TransferActivity) Execute(ctx context.Context, params *TransferActivityParams) (*TransferActivityResponse, error) {
	p, err := a.manager.Pipelines.ByName(params.PipelineName)
	if err != nil {
		return nil, wferrors.NonRetryableError(err)
	}
	amc := p.Client()

	// Transfer path should include the location UUID if defined.
	var path = params.RelPath
	if params.TransferLocationID != "" {
		path = fmt.Sprintf("%s:%s", params.TransferLocationID, path)
	}

	var resp *amclient.PackageCreateResponse
	var httpResp *amclient.Response
	var backoffStrategy = backoff.WithContext(backoff.NewConstantBackOff(25*time.Second), ctx)

	err = backoff.RetryNotify(
		func() (err error) {
			ctx, cancel := context.WithTimeout(ctx, time.Second*10)
			defer cancel()

			resp, httpResp, err = amc.Package.Create(ctx, &amclient.PackageCreateRequest{
				Name:             params.Name,
				Path:             path,
				ProcessingConfig: params.ProcessingConfig,
				AutoApprove:      &params.AutoApprove,
			})
			if err != nil {
				if httpResp != nil {
					switch {
					case httpResp.StatusCode == http.StatusForbidden:
						return backoff.Permanent(wferrors.NonRetryableError(fmt.Errorf("authentication error: %v", err)))
					}
				}
			}

			return err
		},
		backoffStrategy,
		func(err error, duration time.Duration) {
			activity.RecordHeartbeat(ctx, err.Error())
		},
	)

	return &TransferActivityResponse{
		TransferID:      resp.ID,
		PipelineVersion: httpResp.Header.Get("X-Archivematica-Version"),
		PipelineID:      httpResp.Header.Get("X-Archivematica-ID"),
	}, nil
}
