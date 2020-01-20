package workflow

import (
	"context"
	"time"

	"github.com/artefactual-labs/enduro/internal/collection"
	"github.com/artefactual-labs/enduro/internal/workflow/manager"

	"github.com/go-logr/logr"
	"go.uber.org/cadence/activity"
)

type createPackageLocalActivityParams struct {
	OriginalID string
	Status     collection.Status
}

func createPackageLocalActivity(ctx context.Context, logger logr.Logger, colsvc collection.Service, params *createPackageLocalActivityParams) (uint, error) {
	info := activity.GetInfo(ctx)

	col := &collection.Collection{
		WorkflowID: info.WorkflowExecution.ID,
		RunID:      info.WorkflowExecution.RunID,
		OriginalID: params.OriginalID,
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
	info := activity.GetInfo(ctx)

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

func loadConfigLocalActivity(ctx context.Context, m *manager.Manager, pipeline string, tinfo *TransferInfo) (*TransferInfo, error) {
	p, err := m.Pipelines.ByName(pipeline)
	if err != nil {
		m.Logger.Error(err, "Error loading local configuration")
		return nil, err
	}

	tinfo.PipelineConfig = p.Config()
	tinfo.Hooks = m.Hooks

	return tinfo, nil
}
