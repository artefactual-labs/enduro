// Package workflow contains an experimental workflow for Archivemica transfers.
//
// It's not generalized since it contains client-specific activities. However,
// the long-term goal is to build a system where workflows and activities are
// dynamically set up based on user input.
package workflow

import (
	"context"
	"fmt"
	"time"

	"github.com/artefactual-labs/enduro/internal/collection"
	"github.com/artefactual-labs/enduro/internal/pipeline"
	"github.com/artefactual-labs/enduro/internal/watcher"

	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/workflow"
	"go.uber.org/zap"
)

const (
	DownloadActivityName               = "download-activity"
	BundleActivityName                 = "bundle-activity"
	TransferActivityName               = "transfer-activity"
	PollTransferActivityName           = "poll-transfer-activity"
	PollIngestActivityName             = "poll-ingest-activity"
	UpdateHARIActivityName             = "update-hari-activity"
	UpdateProductionSystemActivityName = "update-production-system-activity"
	CleanUpActivityName                = "clean-up-activity"
	HidePackageActivityName            = "hide-package-activity"
	DeleteOriginalActivityName         = "delete-original-activity"
)

type ProcessingWorkflow struct {
	manager *Manager
}

func NewProcessingWorkflow(m *Manager) *ProcessingWorkflow {
	return &ProcessingWorkflow{manager: m}
}

type BundleInfo struct {
}

// TransferInfo is shared state that is passed down to activities. It can be
// useful for hooks that may require quick access to processing state.
// TODO: clean this up, e.g.: it can embed a collection.Collection.
type TransferInfo struct {

	// TempFile is the temporary location where the blob is downloaded.
	//
	// It is populated by the workflow with the result of DownloadActivity.
	TempFile string

	// TransferID given by Archivematica.
	//
	// It is populated by TransferActivity.
	TransferID string

	// SIPID given by Archivematica.
	//
	// It is populated by PollTransferActivity.
	SIPID string

	// Enduro internal collection ID.
	//
	// It is populated via the workflow request or createPackageLocalActivity.
	CollectionID uint

	// Original watcher event.
	//
	// It is populated via the workflow request.
	Event *watcher.BlobEvent

	// OriginalID is the UUID found in the key of the blob. It can be empty.
	//
	// It is populated from the workflow (deterministically).
	OriginalID string

	// Status of the collection.
	//
	// It is populated from the workflow (deterministically)
	Status collection.Status

	// StoredAt is the time when the AIP is stored.
	//
	// It is populated by PollIngestActivity as long as Ingest completes.
	StoredAt time.Time

	// PipelineConfig is the configuration of the pipeline that this workflow
	// uses to provide access to its activities.
	//
	// It is populated by loadConfigLocalActivity.
	PipelineConfig *pipeline.Config

	// Hooks is the hook config store.
	//
	// It is populated by loadConfigLocalActivity.
	Hooks map[string]map[string]interface{}

	// Information about the bundle (transfer) that we submit to Archivematica.
	// Full path, relative path, name, kind...
	//
	// It is populated by BundleActivity.
	Bundle BundleActivityResult
}

// ProcessingWorkflow orchestrates all the activities related to the processing
// of a SIP in Archivematica, including is retrieval, creation of transfer,
// etc...
//
// Retrying this workflow would result in a new Archivematica transfer. We  do
// not have a retry policy in place. The user could trigger a new instance via
// the API.
func (w *ProcessingWorkflow) Execute(ctx workflow.Context, req *collection.ProcessingWorkflowRequest) error {
	tinfo := &TransferInfo{
		CollectionID: req.CollectionID,
		Event:        req.Event,
		OriginalID:   req.Event.NameUUID(),
		Status:       collection.StatusInProgress,
	}

	// Persist collection as early as possible.
	activityOpts := withLocalActivityOpts(ctx)
	err := workflow.ExecuteLocalActivity(activityOpts, createPackageLocalActivity, w.manager.Collection, tinfo).Get(activityOpts, &tinfo)
	if err != nil {
		return nonRetryableError(fmt.Errorf("error persisting collection: %v", err))
	}

	// Load pipeline configuration and hooks.
	activityOpts = withLocalActivityOpts(ctx)
	err = workflow.ExecuteLocalActivity(activityOpts, loadConfigLocalActivity, w.manager, req.Event.PipelineName, tinfo).Get(activityOpts, &tinfo)
	if err != nil {
		return nonRetryableError(fmt.Errorf("error loading configuration: %v", err))
	}

	// A session guarantees that activities within it are scheduled on the same
	// workflow.
	var sessCtx workflow.Context
	var sessErr error
	{
		activityOpts = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			ScheduleToStartTimeout: time.Second * 5,
			StartToCloseTimeout:    time.Minute,
		})
		sessCtx, err = workflow.CreateSession(activityOpts, &workflow.SessionOptions{
			CreationTimeout:  time.Minute,
			ExecutionTimeout: time.Hour * 24 * 5,
		})
		if err != nil {
			return nonRetryableError(fmt.Errorf("error creating session: %w", err))
		}

		sessErr = w.SessionHandler(ctx, sessCtx, tinfo)
	}

	if sessErr != nil {
		tinfo.Status = collection.StatusError
	} else {
		tinfo.Status = collection.StatusDone
	}

	// Update package status.
	var disconnectedCtx, _ = workflow.NewDisconnectedContext(ctx)
	activityOpts = withLocalActivityOpts(disconnectedCtx)
	_ = workflow.ExecuteLocalActivity(activityOpts, updatePackageStatusLocalActivity, w.manager.Collection, tinfo).Get(activityOpts, nil)

	// One of the activities within the session has failed. There's not much we
	// can do if the worker died at this point, since what we aim to do next
	// depends on resources only available within that worker.
	if sessErr == workflow.ErrSessionFailed {
		workflow.CompleteSession(sessCtx)
		return sessErr
	}

	// Schedule deletion of the original in the watched data source.
	var deletionTimer workflow.Future
	if tinfo.Status == collection.StatusDone && tinfo.Event.RetentionPeriod != nil {
		deletionTimer = workflow.NewTimer(ctx, *tinfo.Event.RetentionPeriod)
	}

	// Activities that we want to run within the session regardless the
	// result. E.g. receipts, clean-ups, etc...
	// Passing the activity lets the activity determine if the process failed.
	var futures []workflow.Future
	activityOpts = withActivityOptsForRequest(sessCtx)
	if disabled, _ := hookAttrBool(tinfo.Hooks, "hari", "disabled"); !disabled {
		futures = append(futures, workflow.ExecuteActivity(activityOpts, UpdateHARIActivityName, tinfo))
	}
	if disabled, _ := hookAttrBool(tinfo.Hooks, "prod", "disabled"); !disabled {
		futures = append(futures, workflow.ExecuteActivity(activityOpts, UpdateProductionSystemActivityName, tinfo))
	}
	for _, f := range futures {
		_ = f.Get(activityOpts, nil)
	}

	// Hide packages from Archivematica Dashboard.
	if tinfo.Status == collection.StatusDone {
		futures = []workflow.Future{}
		activityOpts = withActivityOptsForRequest(ctx)
		futures = append(futures, workflow.ExecuteActivity(activityOpts, HidePackageActivityName, tinfo.TransferID, "transfer", tinfo.Event.PipelineName))
		futures = append(futures, workflow.ExecuteActivity(activityOpts, HidePackageActivityName, tinfo.SIPID, "ingest", tinfo.Event.PipelineName))
		for _, f := range futures {
			_ = f.Get(activityOpts, nil)
		}
	}

	// This is the last activity that depends on the session.
	activityOpts = withActivityOptsForRequest(sessCtx)
	_ = workflow.ExecuteActivity(activityOpts, CleanUpActivityName, &CleanUpActivityParams{
		FullPath: tinfo.Bundle.FullPath,
	}).Get(activityOpts, nil)

	workflow.CompleteSession(sessCtx)

	var logger = workflow.GetLogger(ctx)

	// Delete original once the timer returns.
	if deletionTimer != nil {
		err := deletionTimer.Get(ctx, nil)
		if err != nil {
			logger.Warn("Retention policy timer failed", zap.Error(err))
		} else {
			activityOpts = withActivityOptsForRequest(ctx)
			_ = workflow.ExecuteActivity(activityOpts, DeleteOriginalActivityName, tinfo.Event).Get(activityOpts, nil)
		}
	}

	logger.Info(
		"Workflow completed successfully!",
		zap.Uint("collectionID", tinfo.CollectionID),
		zap.String("pipeline", tinfo.Event.PipelineName),
		zap.String("event", tinfo.Event.String()),
		zap.String("status", tinfo.Status.String()),
	)

	return nil
}

func (w *ProcessingWorkflow) SessionHandler(ctx workflow.Context, sessCtx workflow.Context, tinfo *TransferInfo) error {
	var (
		activityOpts workflow.Context
		err          error
	)

	// Download.
	activityOpts = withActivityOptsForLongLivedRequest(sessCtx)
	err = workflow.ExecuteActivity(activityOpts, DownloadActivityName, tinfo.Event).Get(activityOpts, &tinfo.TempFile)
	if err != nil {
		return err
	}

	// Bundle.
	activityOpts = withActivityOptsForLongLivedRequest(sessCtx)
	err = workflow.ExecuteActivity(activityOpts, BundleActivityName, &BundleActivityParams{
		TransferDir: tinfo.PipelineConfig.TransferDir,
		Key:         tinfo.Event.Key,
		TempFile:    tinfo.TempFile,
	}).Get(activityOpts, &tinfo.Bundle)
	if err != nil {
		return err
	}

	// Transfer.
	activityOpts = withActivityOptsForRequest(sessCtx)
	err = workflow.ExecuteActivity(activityOpts, TransferActivityName, &TransferActivityParams{
		PipelineName:       tinfo.Event.PipelineName,
		TransferLocationID: tinfo.PipelineConfig.TransferLocationID,
		RelPath:            tinfo.Bundle.RelPath,
		Name:               tinfo.Bundle.Name,
		ProcessingConfig:   tinfo.PipelineConfig.ProcessingConfig,
		AutoApprove:        true,
	}).Get(activityOpts, &tinfo.TransferID)
	if err != nil {
		return err
	}

	// Update status of collection.
	activityOpts = withLocalActivityOpts(ctx)
	_ = workflow.ExecuteLocalActivity(activityOpts, updatePackageStatusLocalActivity, w.manager.Collection, tinfo).Get(activityOpts, nil)

	// Poll transfer.
	activityOpts = withActivityOptsForHeartbeatedRequest(sessCtx, time.Minute)
	err = workflow.ExecuteActivity(activityOpts, PollTransferActivityName, &PollTransferActivityParams{
		PipelineName: tinfo.Event.PipelineName,
		TransferID:   tinfo.TransferID,
	}).Get(activityOpts, &tinfo.SIPID)
	if err != nil {
		return err
	}

	// Poll ingest.
	activityOpts = withActivityOptsForHeartbeatedRequest(sessCtx, time.Minute)
	err = workflow.ExecuteActivity(activityOpts, PollIngestActivityName, &PollIngestActivityParams{
		PipelineName: tinfo.Event.PipelineName,
		SIPID:        tinfo.SIPID,
	}).Get(activityOpts, &tinfo.StoredAt)
	if err != nil {
		return err
	}

	return nil
}

// Local activities.

func createPackageLocalActivity(ctx context.Context, colsvc collection.Service, tinfo *TransferInfo) (*TransferInfo, error) {
	info := activity.GetInfo(ctx)

	if tinfo.CollectionID > 0 {
		err := updatePackageStatusLocalActivity(ctx, colsvc, tinfo)
		return tinfo, err
	}

	col := &collection.Collection{
		WorkflowID: info.WorkflowExecution.ID,
		RunID:      info.WorkflowExecution.RunID,
		OriginalID: tinfo.OriginalID,
		Status:     tinfo.Status,
	}

	if err := colsvc.Create(ctx, col); err != nil {
		return tinfo, err
	}

	tinfo.CollectionID = col.ID

	return tinfo, nil
}

func updatePackageStatusLocalActivity(ctx context.Context, colsvc collection.Service, tinfo *TransferInfo) error {
	info := activity.GetInfo(ctx)

	return colsvc.UpdateWorkflowStatus(
		ctx, tinfo.CollectionID, tinfo.Bundle.Name, info.WorkflowExecution.ID,
		info.WorkflowExecution.RunID, tinfo.TransferID, tinfo.SIPID,
		tinfo.Status, tinfo.StoredAt,
	)
}

func loadConfigLocalActivity(ctx context.Context, m *Manager, pipeline string, tinfo *TransferInfo) (*TransferInfo, error) {
	p, err := m.Pipelines.Pipeline(pipeline)
	if err != nil {
		return nil, err
	}

	tinfo.PipelineConfig = p.Config()
	tinfo.Hooks = m.Hooks

	return tinfo, nil
}
