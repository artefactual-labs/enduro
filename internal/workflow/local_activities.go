package workflow

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	temporalsdk_activity "go.temporal.io/sdk/activity"

	"github.com/artefactual-labs/enduro/internal/collection"
	"github.com/artefactual-labs/enduro/internal/pipeline"
	"github.com/artefactual-labs/enduro/internal/workflow/hooks"
)

type createPackageLocalActivityParams struct {
	Key    string
	Status collection.Status
}

func createPackageLocalActivity(ctx context.Context, logger logr.Logger, colsvc collection.Service, params *createPackageLocalActivityParams) (uint, error) {
	info := temporalsdk_activity.GetInfo(ctx)

	col := &collection.Collection{
		Name:       params.Key,
		WorkflowID: info.WorkflowExecution.ID,
		RunID:      info.WorkflowExecution.RunID,
		Status:     params.Status,
	}

	if err := colsvc.Create(ctx, col); err != nil {
		logger.Error(err, "Error creating collection")
		return 0, err
	}

	return col.ID, nil
}

type updatePackageLocalActivityParams struct {
	CollectionID uint
	Key          string
	PipelineID   string
	TransferID   string
	SIPID        string
	StoredAt     time.Time
	Status       collection.Status
}

func updatePackageLocalActivity(ctx context.Context, logger logr.Logger, colsvc collection.Service, params *updatePackageLocalActivityParams) error {
	info := temporalsdk_activity.GetInfo(ctx)

	err := colsvc.UpdateWorkflowStatus(
		ctx, params.CollectionID, params.Key, info.WorkflowExecution.ID,
		info.WorkflowExecution.RunID, params.TransferID, params.SIPID, params.PipelineID,
		params.Status, params.StoredAt,
	)
	if err != nil {
		logger.Error(err, "Error updating collection")
		return err
	}

	return nil
}

func checkDuplicatePackageLocalActivity(ctx context.Context, logger logr.Logger, colsvc collection.Service, id uint) (bool, error) {
	return colsvc.CheckDuplicate(ctx, id)
}

func loadConfigLocalActivity(ctx context.Context, h *hooks.Hooks, pipelineRegistry *pipeline.Registry, logger logr.Logger, pipeline string, tinfo *TransferInfo) (*TransferInfo, error) {
	p, err := pipelineRegistry.ByName(pipeline)
	if err != nil {
		logger.Error(err, "Error loading local configuration")
		return nil, err
	}

	tinfo.PipelineConfig = p.Config()
	tinfo.Hooks = h.Hooks

	return tinfo, nil
}

func setStatusInProgressLocalActivity(ctx context.Context, colsvc collection.Service, colID uint, startedAt time.Time) error {
	return colsvc.SetStatusInProgress(ctx, colID, startedAt)
}

//nolint:unused
func setStatusLocalActivity(ctx context.Context, colsvc collection.Service, colID uint, status collection.Status) error {
	return colsvc.SetStatus(ctx, colID, status)
}

func setOriginalIDLocalActivity(ctx context.Context, colsvc collection.Service, colID uint, originalID string) error {
	return colsvc.SetOriginalID(ctx, colID, originalID)
}
