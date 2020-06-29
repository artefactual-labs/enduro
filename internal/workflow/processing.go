// Package workflow contains an experimental workflow for Archivemica transfers.
//
// It's not generalized since it contains client-specific activities. However,
// the long-term goal is to build a system where workflows and activities are
// dynamically set up based on user input.
package workflow

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/artefactual-labs/enduro/internal/collection"
	"github.com/artefactual-labs/enduro/internal/nha"
	nha_activities "github.com/artefactual-labs/enduro/internal/nha/activities"
	"github.com/artefactual-labs/enduro/internal/pipeline"
	"github.com/artefactual-labs/enduro/internal/validation"
	"github.com/artefactual-labs/enduro/internal/watcher"
	"github.com/artefactual-labs/enduro/internal/workflow/activities"
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

	// Pipeline name.
	//
	// It is populated via the workflow request.
	PipelineName string

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
}

// ProcessingWorkflow orchestrates all the activities related to the processing
// of a SIP in Archivematica, including is retrieval, creation of transfer,
// etc...
//
// Retrying this workflow would result in a new Archivematica transfer. We  do
// not have a retry policy in place. The user could trigger a new instance via
// the API.
func (w *ProcessingWorkflow) Execute(ctx workflow.Context, req *collection.ProcessingWorkflowRequest) error {
	var (
		logger = workflow.GetLogger(ctx)

		tinfo = &TransferInfo{
			CollectionID: req.CollectionID,
			Event:        req.Event,
			PipelineName: req.PipelineName,
		}

		// Attributes inferred from the name of the transfer. Populated by parseNameLocalActivity.
		nameInfo nha.NameInfo

		// Collection status. All collections start in queued status.
		status = collection.StatusQueued
	)

	// Persist collection as early as possible.
	{
		var activityOpts = withLocalActivityOpts(ctx)
		var err error

		if req.CollectionID == 0 {
			err = workflow.ExecuteLocalActivity(activityOpts, createPackageLocalActivity, w.manager.Logger, w.manager.Collection, &createPackageLocalActivityParams{
				Key:    req.Event.Key,
				Status: status,
			}).Get(activityOpts, &tinfo.CollectionID)
		} else {
			// TODO: investigate better way to reset the collection.
			err = workflow.ExecuteLocalActivity(activityOpts, updatePackageLocalActivity, w.manager.Logger, w.manager.Collection, &updatePackageLocalActivityParams{
				CollectionID: req.CollectionID,
				Key:          req.Event.Key,
				PipelineID:   "",
				TransferID:   "",
				SIPID:        "",
				StoredAt:     time.Time{},
				Status:       status,
			}).Get(activityOpts, nil)
		}

		if err != nil {
			return fmt.Errorf("error persisting collection: %v", err)
		}
	}

	// Ensure that the status of the collection is always updated when this
	// workflow function returns.
	defer func() {
		// Mark as failed unless it completed successfully or it was abandoned.
		if status != collection.StatusDone && status != collection.StatusAbandoned {
			status = collection.StatusError
		}

		// Use disconnected context so it also runs after cancellation.
		var dctx, _ = workflow.NewDisconnectedContext(ctx)
		var activityOpts = withLocalActivityOpts(dctx)
		_ = workflow.ExecuteLocalActivity(activityOpts, updatePackageLocalActivity, w.manager.Logger, w.manager.Collection, &updatePackageLocalActivityParams{
			CollectionID: tinfo.CollectionID,
			Key:          tinfo.Event.Key,
			PipelineID:   tinfo.PipelineID,
			TransferID:   tinfo.TransferID,
			SIPID:        tinfo.SIPID,
			StoredAt:     tinfo.StoredAt,
			Status:       status,
		}).Get(activityOpts, nil)
	}()

	// Extract details from transfer name.
	{
		var activityOpts = withLocalActivityWithoutRetriesOpts(ctx)
		err := workflow.ExecuteLocalActivity(activityOpts, nha_activities.ParseNameLocalActivity, req.Event.Key).Get(activityOpts, &nameInfo)

		// An error should only stop the workflow if hari/prod activities are enabled.
		hariDisabled, _ := manager.HookAttrBool(w.manager.Hooks, "hari", "disabled")
		prodDisabled, _ := manager.HookAttrBool(w.manager.Hooks, "prod", "disabled")
		if err != nil && !hariDisabled && !prodDisabled {
			return fmt.Errorf("error parsing transfer name: %v", err)
		}

		if nameInfo.Identifier != "" {
			activityOpts = withLocalActivityOpts(ctx)
			_ = workflow.ExecuteLocalActivity(activityOpts, setOriginalIDLocalActivity, w.manager.Collection, tinfo.CollectionID, nameInfo.Identifier).Get(activityOpts, nil)
		}
	}

	// Load pipeline configuration and hooks.
	{
		activityOpts := withLocalActivityWithoutRetriesOpts(ctx)
		err := workflow.ExecuteLocalActivity(activityOpts, loadConfigLocalActivity, w.manager, tinfo.PipelineName, tinfo).Get(activityOpts, &tinfo)
		if err != nil {
			return fmt.Errorf("error loading configuration: %v", err)
		}
	}

	// Activities running within a session.
	{
		var sessErr error
		var maxAttempts = 5

		for attempt := 1; attempt <= maxAttempts; attempt++ {
			activityOpts := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
				ScheduleToStartTimeout: forever,
				StartToCloseTimeout:    time.Minute,
			})
			sessCtx, err := workflow.CreateSession(activityOpts, &workflow.SessionOptions{
				CreationTimeout:  forever,
				ExecutionTimeout: forever,
			})
			if err != nil {
				return fmt.Errorf("error creating session: %v", err)
			}

			sessErr = w.SessionHandler(sessCtx, attempt, tinfo, nameInfo, req.ValidationConfig)

			// We want to retry the session if it has been canceled as a result
			// of losing the worker but not otherwise. This scenario seems to be
			// identifiable when we have an error but the root context has not
			// been canceled.
			if sessErr != nil && (errors.Is(sessErr, workflow.ErrSessionFailed) || errors.Is(sessErr, workflow.ErrCanceled) || strings.Contains(sessErr.Error(), "CanceledError")) {
				// Root context canceled, hence workflow canceled.
				if ctx.Err() == workflow.ErrCanceled {
					return nil
				}

				logger.Error("Session failed, will retry shortly (10s)...",
					zap.NamedError("rootCtx", ctx.Err()),
					zap.Int("attemptFailed", attempt),
					zap.Int("attemptsLeft", maxAttempts-attempt))

				_ = workflow.Sleep(ctx, time.Second*10)

				continue
			}
			break
		}

		if sessErr != nil {
			status = collection.StatusError

			if errors.Is(sessErr, ErrAsyncCompletionAbandoned) {
				status = collection.StatusAbandoned
			}

			return sessErr
		}

		status = collection.StatusDone
	}

	// Hide packages from Archivematica Dashboard.
	{
		if status == collection.StatusDone {
			futures := []workflow.Future{}
			activityOpts := withActivityOptsForRequest(ctx)
			futures = append(futures, workflow.ExecuteActivity(activityOpts, activities.HidePackageActivityName, tinfo.TransferID, "transfer", tinfo.PipelineName))
			futures = append(futures, workflow.ExecuteActivity(activityOpts, activities.HidePackageActivityName, tinfo.SIPID, "ingest", tinfo.PipelineName))
			for _, f := range futures {
				_ = f.Get(activityOpts, nil)
			}
		}
	}

	// Schedule deletion of the original in the watched data source.
	{
		if status == collection.StatusDone && tinfo.Event.RetentionPeriod != nil {
			err := workflow.NewTimer(ctx, *tinfo.Event.RetentionPeriod).Get(ctx, nil)
			if err != nil {
				logger.Warn("Retention policy timer failed", zap.Error(err))
			} else {
				activityOpts := withActivityOptsForRequest(ctx)
				_ = workflow.ExecuteActivity(activityOpts, activities.DeleteOriginalActivityName, tinfo.Event.WatcherName, tinfo.Event.Key).Get(activityOpts, nil)
			}
		}
	}

	logger.Info(
		"Workflow completed successfully!",
		zap.Uint("collectionID", tinfo.CollectionID),
		zap.String("pipeline", tinfo.PipelineName),
		zap.String("event", tinfo.Event.String()),
		zap.String("status", status.String()),
	)

	return nil
}

// SessionHandler runs activities that belong to the same session.
func (w *ProcessingWorkflow) SessionHandler(sessCtx workflow.Context, attempt int, tinfo *TransferInfo, nameInfo nha.NameInfo, validationConfig validation.Config) error {
	defer func() {
		_ = releasePipeline(sessCtx, w.manager, tinfo.PipelineName)
		workflow.CompleteSession(sessCtx)
	}()

	// Block until pipeline semaphore is acquired. The collection status is set
	// to in-progress as soon as the operation succeeds.
	{
		if err := acquirePipeline(sessCtx, w.manager, tinfo.PipelineName, tinfo.CollectionID); err != nil {
			return err
		}
	}

	// Download.
	{
		// TODO: even if TempFile is defined, we should confirm that the file is
		// locally available in disk, just in case we're in the context of a
		// session retry where a different working is doing the work. In that
		// case, the activity whould be executed again.
		if tinfo.TempFile == "" {
			activityOpts := withActivityOptsForLongLivedRequest(sessCtx)
			err := workflow.ExecuteActivity(activityOpts, activities.DownloadActivityName, tinfo.PipelineName, tinfo.Event.WatcherName, tinfo.Event.Key).Get(activityOpts, &tinfo.TempFile)
			if err != nil {
				return err
			}
		}
	}

	// Bundle.
	{
		if tinfo.Bundle == (activities.BundleActivityResult{}) {
			activityOpts := withActivityOptsForLongLivedRequest(sessCtx)
			err := workflow.ExecuteActivity(activityOpts, activities.BundleActivityName, &activities.BundleActivityParams{
				TransferDir:      tinfo.PipelineConfig.TransferDir,
				Key:              tinfo.Event.Key,
				TempFile:         tinfo.TempFile,
				StripTopLevelDir: tinfo.Event.StripTopLevelDir,
			}).Get(activityOpts, &tinfo.Bundle)
			if err != nil {
				return err
			}
		}
	}

	// Validate transfer.
	{
		if validationConfig.IsEnabled() && tinfo.Bundle != (activities.BundleActivityResult{}) {
			activityOpts := workflow.WithActivityOptions(sessCtx, workflow.ActivityOptions{
				ScheduleToStartTimeout: forever,
				StartToCloseTimeout:    time.Minute * 5,
			})
			err := workflow.ExecuteActivity(activityOpts, activities.ValidateTransferActivityName, &activities.ValidateTransferActivityParams{
				Config: validationConfig,
				Path:   tinfo.Bundle.FullPath,
			}).Get(activityOpts, nil)
			if err != nil {
				return err
			}
		}
	}

	// Transfer.
	{
		if tinfo.TransferID == "" {
			var transferResponse = activities.TransferActivityResponse{}

			activityOpts := withActivityOptsForRequest(sessCtx)
			err := workflow.ExecuteActivity(activityOpts, activities.TransferActivityName, &activities.TransferActivityParams{
				PipelineName:       tinfo.PipelineName,
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
		}
	}

	// Persist TransferID + PipelineID.
	{
		activityOpts := withLocalActivityOpts(sessCtx)
		_ = workflow.ExecuteLocalActivity(activityOpts, updatePackageLocalActivity, w.manager.Logger, w.manager.Collection, &updatePackageLocalActivityParams{
			CollectionID: tinfo.CollectionID,
			Key:          tinfo.Event.Key,
			Status:       collection.StatusInProgress,
			TransferID:   tinfo.TransferID,
			PipelineID:   tinfo.PipelineID,
		}).Get(activityOpts, nil)
	}

	// Poll transfer.
	{
		if tinfo.SIPID == "" {
			activityOpts := withActivityOptsForHeartbeatedRequest(sessCtx, time.Minute)
			err := workflow.ExecuteActivity(activityOpts, activities.PollTransferActivityName, &activities.PollTransferActivityParams{
				PipelineName: tinfo.PipelineName,
				TransferID:   tinfo.TransferID,
			}).Get(activityOpts, &tinfo.SIPID)
			if err != nil {
				return err
			}
		}
	}

	// Persist SIPID.
	{
		activityOpts := withLocalActivityOpts(sessCtx)
		_ = workflow.ExecuteLocalActivity(activityOpts, updatePackageLocalActivity, w.manager.Logger, w.manager.Collection, &updatePackageLocalActivityParams{
			CollectionID: tinfo.CollectionID,
			Key:          tinfo.Event.Key,
			TransferID:   tinfo.TransferID,
			PipelineID:   tinfo.PipelineID,
			Status:       collection.StatusInProgress,
			SIPID:        tinfo.SIPID,
		}).Get(activityOpts, nil)
	}

	// Poll ingest.
	{
		if tinfo.StoredAt.IsZero() {
			activityOpts := withActivityOptsForHeartbeatedRequest(sessCtx, time.Minute)
			err := workflow.ExecuteActivity(activityOpts, activities.PollIngestActivityName, &activities.PollIngestActivityParams{
				PipelineName: tinfo.PipelineName,
				SIPID:        tinfo.SIPID,
			}).Get(activityOpts, &tinfo.StoredAt)
			if err != nil {
				return err
			}
		}
	}

	// Deliver receipts.
	{
		err := w.sendReceipts(sessCtx, &sendReceiptsParams{
			SIPID:        tinfo.SIPID,
			StoredAt:     tinfo.StoredAt,
			FullPath:     tinfo.Bundle.FullPath,
			PipelineName: tinfo.PipelineName,
			NameInfo:     nameInfo,
			CollectionID: tinfo.CollectionID,
		})
		if err != nil {
			return fmt.Errorf("error delivering receipt(s): %w", err)
		}
	}

	// Delete local temporary files.
	{
		if tinfo.Bundle.FullPathBeforeStrip != "" {
			activityOpts := withActivityOptsForRequest(sessCtx)
			_ = workflow.ExecuteActivity(activityOpts, activities.CleanUpActivityName, &activities.CleanUpActivityParams{
				FullPath: tinfo.Bundle.FullPathBeforeStrip,
			}).Get(activityOpts, nil)
		}
	}

	return nil
}
