package workflow

import (
	"fmt"
	"time"

	"github.com/artefactual-labs/enduro/internal/nha"
	nha_activities "github.com/artefactual-labs/enduro/internal/nha/activities"
	"github.com/artefactual-labs/enduro/internal/workflow/manager"

	"go.uber.org/cadence/workflow"
)

type sendReceiptsParams struct {
	SIPID        string
	StoredAt     time.Time
	FullPath     string
	PipelineName string
	NameInfo     nha.NameInfo
}

func sendReceipts(ctx workflow.Context, hooks map[string]map[string]interface{}, params *sendReceiptsParams) error {
	ctx = withActivityOptsForRequest(ctx)

	if disabled, _ := manager.HookAttrBool(hooks, "hari", "disabled"); !disabled {
		err := workflow.ExecuteActivity(ctx, nha_activities.UpdateHARIActivityName, &nha_activities.UpdateHARIActivityParams{
			SIPID:        params.SIPID,
			StoredAt:     params.StoredAt,
			FullPath:     params.FullPath,
			PipelineName: params.PipelineName,
			NameInfo:     params.NameInfo,
		}).Get(ctx, nil)

		if err != nil {
			return fmt.Errorf("error sending hari receipt: %v", err)
		}
	}

	if disabled, _ := manager.HookAttrBool(hooks, "prod", "disabled"); !disabled {
		err := workflow.ExecuteActivity(ctx, nha_activities.UpdateProductionSystemActivityName, &nha_activities.UpdateProductionSystemActivityParams{
			StoredAt:     params.StoredAt,
			PipelineName: params.PipelineName,
			NameInfo:     params.NameInfo,
		}).Get(ctx, nil)

		if err != nil {
			return fmt.Errorf("error sending prod receipt: %v", err)
		}
	}

	return nil
}
