package workflow

import (
	"context"

	"github.com/artefactual-labs/enduro/internal/collection"
	"github.com/artefactual-labs/enduro/internal/pipeline"
	"github.com/artefactual-labs/enduro/internal/workflow/manager"

	"go.uber.org/cadence/activity"
)

func releasePipelineLocalActivity(ctx context.Context, registry *pipeline.Registry, name string) error {
	p, err := registry.ByName(name)
	if err != nil {
		return err
	}

	p.Release()

	return nil
}

func createPackageLocalActivity(ctx context.Context, colsvc collection.Service, tinfo *TransferInfo) (*TransferInfo, error) {
	info := activity.GetInfo(ctx)

	if tinfo.CollectionID > 0 {
		err := updatePackageStatusLocalActivity(ctx, colsvc, tinfo, tinfo.Status)
		return tinfo, err
	}

	col := &collection.Collection{
		WorkflowID: info.WorkflowExecution.ID,
		RunID:      info.WorkflowExecution.RunID,
		OriginalID: tinfo.NameInfo.Identifier,
		Status:     tinfo.Status,
	}

	if err := colsvc.Create(ctx, col); err != nil {
		return tinfo, err
	}

	tinfo.CollectionID = col.ID

	return tinfo, nil
}

func updatePackageStatusLocalActivity(ctx context.Context, colsvc collection.Service, tinfo *TransferInfo, status collection.Status) error {
	info := activity.GetInfo(ctx)

	return colsvc.UpdateWorkflowStatus(
		ctx, tinfo.CollectionID, tinfo.Event.Key, info.WorkflowExecution.ID,
		info.WorkflowExecution.RunID, tinfo.TransferID, tinfo.SIPID, tinfo.PipelineID,
		status, tinfo.StoredAt,
	)
}

func loadConfigLocalActivity(ctx context.Context, m *manager.Manager, pipeline string, tinfo *TransferInfo) (*TransferInfo, error) {
	p, err := m.Pipelines.ByName(pipeline)
	if err != nil {
		return nil, err
	}

	tinfo.PipelineConfig = p.Config()
	tinfo.Hooks = m.Hooks

	return tinfo, nil
}
