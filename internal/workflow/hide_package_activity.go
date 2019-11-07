package workflow

import (
	"context"
	"fmt"
)

type HidePackageActivity struct {
	manager *Manager
}

func NewHidePackageActivity(m *Manager) *HidePackageActivity {
	return &HidePackageActivity{manager: m}
}

func (a *HidePackageActivity) Execute(ctx context.Context, unitID, unitType, pipelineName string) error {
	amc, err := a.manager.Pipelines.Client(pipelineName)
	if err != nil {
		return nonRetryableError(fmt.Errorf("error looking up pipeline config: %v", err))
	}

	if unitType != "transfer" && unitType != "ingest" {
		return nonRetryableError(fmt.Errorf("unexpected unit type: %s", unitType))
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
