package activities

import (
	"context"
	"fmt"
	"net/http"

	"go.artefactual.dev/amclient"

	"github.com/artefactual-labs/enduro/internal/pipeline"
	"github.com/artefactual-labs/enduro/internal/temporal"
)

// HidePackageActivity hides processed packages from Archivematica dashboards.
type HidePackageActivity struct {
	pipelineRegistry *pipeline.Registry
}

func NewHidePackageActivity(pipelineRegistry *pipeline.Registry) *HidePackageActivity {
	return &HidePackageActivity{pipelineRegistry: pipelineRegistry}
}

func (a *HidePackageActivity) Execute(ctx context.Context, unitID, unitType, pipelineName string, ignoreConflict bool) error {
	p, err := a.pipelineRegistry.ByName(pipelineName)
	if err != nil {
		return temporal.NewNonRetryableError(err)
	}
	amc := p.Client()

	if unitType != "transfer" && unitType != "ingest" {
		return temporal.NewNonRetryableError(fmt.Errorf("unexpected unit type: %s", unitType))
	}

	if unitType == "transfer" {
		resp, _, err := amc.Transfer.Hide(ctx, unitID)
		if err != nil {
			if ignoreConflict && hideConflict(err) {
				return nil
			}
			return fmt.Errorf("error hiding transfer: %v", err)
		}
		if !resp.Removed {
			return fmt.Errorf("error hiding transfer: not removed")
		}
	}

	if unitType == "ingest" {
		resp, _, err := amc.Ingest.Hide(ctx, unitID)
		if err != nil {
			if ignoreConflict && hideConflict(err) {
				return nil
			}
			return fmt.Errorf("error hiding sip: %v", err)
		}
		if !resp.Removed {
			return fmt.Errorf("error hiding sip: not removed")
		}
	}

	return nil
}

func hideConflict(err error) bool {
	amErr, ok := err.(*amclient.ErrorResponse)
	return ok && amErr.Response != nil && amErr.Response.StatusCode == http.StatusConflict
}
