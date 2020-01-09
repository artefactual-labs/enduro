// Package workflow contains an experimental workflow for Archivemica transfers.
//
// It's not generalized since it contains client-specific activities. However,
// the long-term goal is to build a system where workflows and activities are
// dynamically set up based on user input.
package workflow

import (
	"errors"
	"fmt"
	"time"

	"github.com/artefactual-labs/enduro/internal/collection"
	"github.com/artefactual-labs/enduro/internal/nha"
	nha_activities "github.com/artefactual-labs/enduro/internal/nha/activities"
	"github.com/artefactual-labs/enduro/internal/pipeline"
	"github.com/artefactual-labs/enduro/internal/watcher"
	"github.com/artefactual-labs/enduro/internal/workflow/activities"
	wferrors "github.com/artefactual-labs/enduro/internal/workflow/errors"
	"github.com/artefactual-labs/enduro/internal/workflow/manager"
	"go.uber.org/cadence/workflow"
	"go.uber.org/zap"
)

type ProcessingWorkflow struct {
	manager *manager.Manager
}

func NewProcessingWorkflow(m *manager.Manager) *ProcessingWorkflow {
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

	// PipelineID is the UUID of the Archivematica pipeline. Extracted from
	// the API response header when the transfer is submitted.
	//
	// It is populated by transferActivity.
	PipelineID string

	// Hooks is the hook config store.
	//
	// It is populated by loadConfigLocalActivity.
	Hooks map[string]map[string]interface{}

	// Information about the bundle (transfer) that we submit to Archivematica.
	// Full path, relative path...
	//
	// It is populated by BundleActivity.
	Bundle activities.BundleActivityResult

	// Aditional attributes inferred from the transfer name.
	//
	// It is populated by parseNameLocalActivity.
	NameInfo nha.NameInfo
}

// ProcessingWorkflow orchestrates all the activities related to the processing
// of a SIP in Archivematica, including is retrieval, creation of transfer,
// etc...
//
// Retrying this workflow would result in a new Archivematica transfer. We  do
// not have a retry policy in place. The user could trigger a new instance via
// the API.
func (w *ProcessingWorkflow) Execute(ctx workflow.Context, req *collection.ProcessingWorkflowRequest) error {
	logger := workflow.GetLogger(ctx)
	tinfo := &TransferInfo{
		CollectionID: req.CollectionID,
		Event:        req.Event,
		Status:       collection.StatusInProgress,
	}

	// Extract details from transfer name.
	activityOpts := withLocalActivityOpts(ctx)
	err := workflow.ExecuteLocalActivity(activityOpts, nha_activities.ParseNameLocalActivity, req.Event.Key).Get(activityOpts, &tinfo.NameInfo)
	if err != nil {
		return wferrors.NonRetryableError(fmt.Errorf("error parsing name: %v", err))
	}

	// Persist collection as early as possible.
	activityOpts = withLocalActivityOpts(ctx)
	err = workflow.ExecuteLocalActivity(activityOpts, createPackageLocalActivity, w.manager.Collection, tinfo).Get(activityOpts, &tinfo)
	if err != nil {
		return wferrors.NonRetryableError(fmt.Errorf("error persisting collection: %v", err))
	}

	defer func() {
		// Update package status, using a workflow-disconnected context to
		// ensure that it runs even after cancellation.
		var status = tinfo.Status
		if status == collection.StatusInProgress {
			status = collection.StatusError
		}
		var dctx, _ = workflow.NewDisconnectedContext(ctx)
		_ = workflow.ExecuteLocalActivity(withLocalActivityOpts(dctx), updatePackageStatusLocalActivity, w.manager.Collection, tinfo, status).Get(activityOpts, nil)
	}()

	// Load pipeline configuration and hooks.
	activityOpts = withLocalActivityOpts(ctx)
	err = workflow.ExecuteLocalActivity(activityOpts, loadConfigLocalActivity, w.manager, req.Event.PipelineName, tinfo).Get(activityOpts, &tinfo)
	if err != nil {
		return wferrors.NonRetryableError(fmt.Errorf("error loading configuration: %v", err))
	}

	// A session guarantees that activities within it are scheduled on the same
	// worker.
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
			return wferrors.NonRetryableError(fmt.Errorf("error creating session: %w", err))
		}

		sessErr = w.SessionHandler(ctx, sessCtx, tinfo)
	}

	if sessErr != nil {
		tinfo.Status = collection.StatusError
	} else {
		tinfo.Status = collection.StatusDone
	}

	// One of the activities within the session has failed. There's not much we
	// can do if the worker died at this point, since what we aim to do next
	// depends on resources only available within that worker.
	if sessErr == workflow.ErrSessionFailed {
		workflow.CompleteSession(sessCtx)
		return sessErr
	}

	// Activities that we want to run within the session regardless the
	// result. E.g. receipts, clean-ups, etc...
	// Passing the activity lets the activity determine if the process failed.
	var futures []workflow.Future
	var receiptsFailed bool
	activityOpts = withActivityOptsForRequest(sessCtx)
	if tinfo.Status == collection.StatusDone {
		if disabled, _ := manager.HookAttrBool(tinfo.Hooks, "hari", "disabled"); !disabled {
			futures = append(futures, workflow.ExecuteActivity(activityOpts, nha_activities.UpdateHARIActivityName, &nha_activities.UpdateHARIActivityParams{
				SIPID:        tinfo.SIPID,
				StoredAt:     tinfo.StoredAt,
				FullPath:     tinfo.Bundle.FullPath,
				PipelineName: tinfo.Event.PipelineName,
				NameInfo:     tinfo.NameInfo,
			}))
		}
	}
	if disabled, _ := manager.HookAttrBool(tinfo.Hooks, "prod", "disabled"); !disabled {
		futures = append(futures, workflow.ExecuteActivity(activityOpts, nha_activities.UpdateProductionSystemActivityName, &nha_activities.UpdateProductionSystemActivityParams{
			StoredAt:     tinfo.StoredAt,
			PipelineName: tinfo.Event.PipelineName,
			Status:       tinfo.Status,
			NameInfo:     tinfo.NameInfo,
		}))
	}
	for _, f := range futures {
		if err := f.Get(activityOpts, nil); err != nil {
			receiptsFailed = true
		}
	}
	// This causes the workflow to fail when hooks are errorful. In the future,
	// we'd prefer to give the user to decide what to do next, e.g. start over,
	// retry hooks individually, etc...
	if receiptsFailed {
		tinfo.Status = collection.StatusError
		return errors.New("at least one hook/receipt activity failed")
	}

	// Clean-up is the last activity that depends on the session.
	// We'll close it as soon as the activity completes.
	if tinfo.Bundle.FullPathBeforeStrip != "" {
		activityOpts = withActivityOptsForRequest(sessCtx)
		_ = workflow.ExecuteActivity(activityOpts, activities.CleanUpActivityName, &activities.CleanUpActivityParams{
			FullPath: tinfo.Bundle.FullPathBeforeStrip,
		}).Get(activityOpts, nil)
	}

	workflow.CompleteSession(sessCtx)

	// Hide packages from Archivematica Dashboard.
	if tinfo.Status == collection.StatusDone {
		futures = []workflow.Future{}
		activityOpts = withActivityOptsForRequest(ctx)
		futures = append(futures, workflow.ExecuteActivity(activityOpts, activities.HidePackageActivityName, tinfo.TransferID, "transfer", tinfo.Event.PipelineName))
		futures = append(futures, workflow.ExecuteActivity(activityOpts, activities.HidePackageActivityName, tinfo.SIPID, "ingest", tinfo.Event.PipelineName))
		for _, f := range futures {
			_ = f.Get(activityOpts, nil)
		}
	}

	// Schedule deletion of the original in the watched data source.
	if tinfo.Status == collection.StatusDone && tinfo.Event.RetentionPeriod != nil {
		err := workflow.NewTimer(ctx, *tinfo.Event.RetentionPeriod).Get(ctx, nil)
		if err != nil {
			logger.Warn("Retention policy timer failed", zap.Error(err))
		} else {
			activityOpts = withActivityOptsForRequest(ctx)
			_ = workflow.ExecuteActivity(activityOpts, activities.DeleteOriginalActivityName, tinfo.Event).Get(activityOpts, nil)
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
	err = workflow.ExecuteActivity(activityOpts, activities.DownloadActivityName, tinfo.Event).Get(activityOpts, &tinfo.TempFile)
	if err != nil {
		return err
	}

	// Bundle.
	activityOpts = withActivityOptsForLongLivedRequest(sessCtx)
	err = workflow.ExecuteActivity(activityOpts, activities.BundleActivityName, &activities.BundleActivityParams{
		TransferDir:      tinfo.PipelineConfig.TransferDir,
		Key:              tinfo.Event.Key,
		TempFile:         tinfo.TempFile,
		StripTopLevelDir: tinfo.Event.StripTopLevelDir,
	}).Get(activityOpts, &tinfo.Bundle)
	if err != nil {
		return err
	}

	// Transfer.
	activityOpts = withActivityOptsForRequest(sessCtx)
	var transferResponse = activities.TransferActivityResponse{}
	err = workflow.ExecuteActivity(activityOpts, activities.TransferActivityName, &activities.TransferActivityParams{
		PipelineName:       tinfo.Event.PipelineName,
		TransferLocationID: tinfo.PipelineConfig.TransferLocationID,
		RelPath:            tinfo.Bundle.RelPath,
		Name:               tinfo.Event.Key,
		ProcessingConfig:   tinfo.PipelineConfig.ProcessingConfig,
		AutoApprove:        true,
	}).Get(activityOpts, &transferResponse)
	if err != nil {
		return err
	}
	tinfo.TransferID = transferResponse.TransferID
	tinfo.PipelineID = transferResponse.PipelineID

	// Update status of collection.
	activityOpts = withLocalActivityOpts(ctx)
	_ = workflow.ExecuteLocalActivity(activityOpts, updatePackageStatusLocalActivity, w.manager.Collection, tinfo, tinfo.Status).Get(activityOpts, nil)

	// Poll transfer.
	activityOpts = withActivityOptsForHeartbeatedRequest(sessCtx, time.Minute)
	err = workflow.ExecuteActivity(activityOpts, activities.PollTransferActivityName, &activities.PollTransferActivityParams{
		PipelineName: tinfo.Event.PipelineName,
		TransferID:   tinfo.TransferID,
	}).Get(activityOpts, &tinfo.SIPID)
	if err != nil {
		return err
	}

	// Poll ingest.
	activityOpts = withActivityOptsForHeartbeatedRequest(sessCtx, time.Minute)
	err = workflow.ExecuteActivity(activityOpts, activities.PollIngestActivityName, &activities.PollIngestActivityParams{
		PipelineName: tinfo.Event.PipelineName,
		SIPID:        tinfo.SIPID,
	}).Get(activityOpts, &tinfo.StoredAt)
	if err != nil {
		return err
	}

	return nil
}
