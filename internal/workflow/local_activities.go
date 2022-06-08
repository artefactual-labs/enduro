package workflow

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	temporalsdk_activity "go.temporal.io/sdk/activity"

	"github.com/artefactual-labs/enduro/internal/collection"
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
	SIPID        string
	StoredAt     time.Time
	Status       collection.Status
}

func updatePackageLocalActivity(ctx context.Context, logger logr.Logger, colsvc collection.Service, params *updatePackageLocalActivityParams) error {
	info := temporalsdk_activity.GetInfo(ctx)

	err := colsvc.UpdateWorkflowStatus(
		ctx,
		params.CollectionID,
		params.Key,
		info.WorkflowExecution.ID,
		info.WorkflowExecution.RunID,
		params.SIPID,
		params.Status,
		params.StoredAt,
	)
	if err != nil {
		logger.Error(err, "Error updating collection")
		return err
	}

	return nil
}

func setStatusInProgressLocalActivity(ctx context.Context, colsvc collection.Service, colID uint, startedAt time.Time) error {
	return colsvc.SetStatusInProgress(ctx, colID, startedAt)
}

//nolint:deadcode,unused
func setStatusLocalActivity(ctx context.Context, colsvc collection.Service, colID uint, status collection.Status) error {
	return colsvc.SetStatus(ctx, colID, status)
}
