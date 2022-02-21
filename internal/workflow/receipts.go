package workflow

import (
	"fmt"
	"time"

	cadencesdk_workflow "go.uber.org/cadence/workflow"

	"github.com/artefactual-labs/enduro/internal/nha"
	nha_activities "github.com/artefactual-labs/enduro/internal/nha/activities"
	"github.com/artefactual-labs/enduro/internal/workflow/manager"
)

type sendReceiptsParams struct {
	SIPID        string
	StoredAt     time.Time
	FullPath     string
	PipelineName string
	NameInfo     nha.NameInfo
	CollectionID uint
}

func (w *ProcessingWorkflow) sendReceipts(ctx cadencesdk_workflow.Context, params *sendReceiptsParams) error {
	if disabled, _ := manager.HookAttrBool(w.manager.Hooks, "hari", "disabled"); !disabled {
		opts := cadencesdk_workflow.ActivityOptions{
			ScheduleToStartTimeout: forever,
			StartToCloseTimeout:    time.Minute * 20,
		}
		err := executeActivityWithAsyncErrorHandling(ctx, w.manager.Collection, params.CollectionID, opts, nha_activities.UpdateHARIActivityName, &nha_activities.UpdateHARIActivityParams{
			SIPID:        params.SIPID,
			StoredAt:     params.StoredAt,
			FullPath:     params.FullPath,
			PipelineName: params.PipelineName,
			NameInfo:     params.NameInfo,
		}).Get(ctx, nil)
		if err != nil {
			return fmt.Errorf("error sending hari receipt: %w", err)
		}
	}

	if disabled, _ := manager.HookAttrBool(w.manager.Hooks, "prod", "disabled"); !disabled {
		opts := cadencesdk_workflow.ActivityOptions{
			ScheduleToStartTimeout: forever,
			StartToCloseTimeout:    time.Second * 10,
		}
		err := executeActivityWithAsyncErrorHandling(ctx, w.manager.Collection, params.CollectionID, opts, nha_activities.UpdateProductionSystemActivityName, &nha_activities.UpdateProductionSystemActivityParams{
			StoredAt:     params.StoredAt,
			PipelineName: params.PipelineName,
			NameInfo:     params.NameInfo,
			FullPath:     params.FullPath,
		}).Get(ctx, nil)
		if err != nil {
			return fmt.Errorf("error sending prod receipt: %w", err)
		}
	}

	return nil
}
