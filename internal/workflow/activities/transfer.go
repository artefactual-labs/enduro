package activities

import (
	"context"
	"fmt"
	"net/http"

	"go.artefactual.dev/amclient"
	temporalsdk_activity "go.temporal.io/sdk/activity"

	"github.com/artefactual-labs/enduro/internal/pipeline"
	"github.com/artefactual-labs/enduro/internal/temporal"
)

// TransferActivity submits the transfer to Archivematica and returns its ID.
//
// This is our first interaction with Archivematica. The workflow ends here
// after authentication errors.
type TransferActivity struct {
	pipelineRegistry *pipeline.Registry
}

func NewTransferActivity(pipelineRegistry *pipeline.Registry) *TransferActivity {
	return &TransferActivity{pipelineRegistry: pipelineRegistry}
}

type TransferActivityParams struct {
	PipelineName       string
	TransferLocationID string
	RelPath            string
	Name               string
	ProcessingConfig   string
	TransferType       string
	Accession          string
}

type TransferActivityResponse struct {
	TransferID      string
	PipelineVersion string
	PipelineID      string
}

func (a *TransferActivity) Execute(ctx context.Context, params *TransferActivityParams) (*TransferActivityResponse, error) {
	p, err := a.pipelineRegistry.ByName(params.PipelineName)
	if err != nil {
		return nil, temporal.NewNonRetryableError(err)
	}
	amc := p.Client()
	logger := temporalsdk_activity.GetLogger(ctx)

	idempotencyKey := ""
	serverInfo, _, err := amc.ServerInfo(ctx)
	if err != nil {
		logger.Warn(
			"Unable to determine Archivematica version; submitting transfer without an idempotency key.",
			"error", err,
		)
	} else if serverInfo.Version.AtLeast(1, 19, 0) {
		idempotencyKey = transferIdempotencyKey(
			temporalsdk_activity.GetInfo(ctx),
		)
	}

	// Transfer path should include the location UUID if defined.
	path := params.RelPath
	if params.TransferLocationID != "" {
		path = fmt.Sprintf("%s:%s", params.TransferLocationID, path)
	}

	autoApprove := true
	resp, httpResp, err := amc.Package.Create(ctx, &amclient.PackageCreateRequest{
		Name:             params.Name,
		Type:             params.TransferType,
		Path:             path,
		ProcessingConfig: params.ProcessingConfig,
		AutoApprove:      &autoApprove,
		Accession:        params.Accession,
		IdempotencyKey:   idempotencyKey,
	})
	if err != nil {
		if httpResp != nil {
			switch httpResp.StatusCode {
			case http.StatusForbidden:
				return nil, temporal.NewNonRetryableError(fmt.Errorf("authentication error: %v", err))
			case http.StatusUnprocessableEntity:
				return nil, temporal.NewNonRetryableError(fmt.Errorf("invalid transfer request: %v", err))
			}
		}
		return nil, err
	}

	return &TransferActivityResponse{
		TransferID:      resp.ID,
		PipelineVersion: httpResp.Header.Get("X-Archivematica-Version"),
		PipelineID:      httpResp.Header.Get("X-Archivematica-ID"),
	}, nil
}

// transferIdempotencyKey returns the identity of the single logical transfer
// submission made by a Temporal workflow run.
//
// Temporal may execute TransferActivity more than once because of its activity
// retry policy or because Enduro recreates a failed worker session. Those
// attempts retain the workflow run ID and must reuse the same key so
// Archivematica can replay the original transfer UUID instead of creating a
// duplicate transfer.
//
// Enduro's operator-triggered Retry behaves differently: it keeps the workflow
// ID but starts a new workflow run, and full reprocessing intentionally submits
// a new Archivematica transfer. A collection ID or workflow ID would therefore
// remain stable for too long, while the run ID matches the intended submission
// boundary.
//
// This assumes that a workflow run submits at most one transfer. If Enduro adds
// multiple submissions per run, workflow-level retries, Continue-As-New, or a
// way to recover an ambiguous submission across operator retries, this key must
// be replaced by a persisted logical submission ID.
func transferIdempotencyKey(info temporalsdk_activity.Info) string {
	runID := info.WorkflowExecution.RunID
	if runID == "" {
		return ""
	}

	return fmt.Sprintf("enduro-transfer-%s", runID)
}
