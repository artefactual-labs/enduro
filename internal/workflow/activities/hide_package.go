package activities

import (
	"context"
	"fmt"

	wferrors "github.com/artefactual-labs/enduro/internal/workflow/errors"
	"github.com/artefactual-labs/enduro/internal/workflow/manager"
)

type HidePackageActivity struct {
	manager *manager.Manager
}

func NewHidePackageActivity(m *manager.Manager) *HidePackageActivity {
	return &HidePackageActivity{manager: m}
}

func (a *HidePackageActivity) Execute(ctx context.Context, unitID, unitType, pipelineName string) error {
	p, err := a.manager.Pipelines.ByName(pipelineName)
	if err != nil {
		return wferrors.NonRetryableError(err)
	}
	amc := p.Client()

	if unitType != "transfer" && unitType != "ingest" {
		return wferrors.NonRetryableError(fmt.Errorf("unexpected unit type: %s", unitType))
	}

	if unitType == "transfer" {
		resp, _, err := amc.Transfer.Hide(ctx, unitID)
		if err != nil {
			return fmt.Errorf("error hiding transfer: %v", err)
		}
		if !resp.Removed {
			return fmt.Errorf("error hiding transfer: not removed")
		}
	}

	if unitType == "ingest" {
		resp, _, err := amc.Ingest.Hide(ctx, unitID)
		if err != nil {
			return fmt.Errorf("error hiding sip: %v", err)
		}
		if !resp.Removed {
			return fmt.Errorf("error hiding sip: not removed")
		}
	}

	return nil
}
