package workflow

import (
	"fmt"
	"time"

	"github.com/artefactual-labs/enduro/internal/nha"
	nha_activities "github.com/artefactual-labs/enduro/internal/nha/activities"
	wferrors "github.com/artefactual-labs/enduro/internal/workflow/errors"
	"github.com/artefactual-labs/enduro/internal/workflow/manager"

	"go.uber.org/cadence"
	"go.uber.org/cadence/workflow"
)

type sendReceiptsParams struct {
	SIPID        string
	StoredAt     time.Time
	FullPath     string
	PipelineName string
	NameInfo     nha.NameInfo
	CollectionID uint
}

func (w *ProcessingWorkflow) sendReceipts(ctx workflow.Context, params *sendReceiptsParams) error {
	if disabled, _ := manager.HookAttrBool(w.manager.Hooks, "hari", "disabled"); !disabled {
		opts := workflow.ActivityOptions{
			ScheduleToStartTimeout: forever,
			StartToCloseTimeout:    time.Second * 10,
			RetryPolicy: &cadence.RetryPolicy{
				InitialInterval:          time.Second,
				BackoffCoefficient:       2,
				MaximumInterval:          time.Minute * 5,
				ExpirationInterval:       time.Minute * 5,
				MaximumAttempts:          20,
				NonRetriableErrorReasons: []string{wferrors.NRE},
			},
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
		opts := workflow.ActivityOptions{
			ScheduleToStartTimeout: forever,
			StartToCloseTimeout:    time.Second * 10,
			RetryPolicy: &cadence.RetryPolicy{
				InitialInterval:          time.Second,
				BackoffCoefficient:       2,
				MaximumInterval:          time.Minute * 5,
				ExpirationInterval:       time.Minute * 5,
				MaximumAttempts:          20,
				NonRetriableErrorReasons: []string{wferrors.NRE},
			},
		}
		err := executeActivityWithAsyncErrorHandling(ctx, w.manager.Collection, params.CollectionID, opts, nha_activities.UpdateProductionSystemActivityName, &nha_activities.UpdateProductionSystemActivityParams{
			StoredAt:     params.StoredAt,
			PipelineName: params.PipelineName,
			NameInfo:     params.NameInfo,
		}).Get(ctx, nil)

		if err != nil {
			return fmt.Errorf("error sending prod receipt: %w", err)
		}
	}

	return nil
}
