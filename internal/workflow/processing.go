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
	"github.com/artefactual-labs/enduro/internal/validation"
	"github.com/artefactual-labs/enduro/internal/workflow/activities"
	"github.com/artefactual-labs/enduro/internal/workflow/manager"

	"go.uber.org/cadence"
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
	// The zero value represents a new collection. It can be used to indicate
	// an existing collection in retries.
	//
	// It is populated via the workflow request or createPackageLocalActivity.
	CollectionID uint

	// Name of the watcher that received this blob.
	//
	// It is populated via the workflow request.
	WatcherName string

	// Pipeline name.
	//
	// It is populated via the workflow request.
	PipelineName string

	// Retention period.
	// Period of time to schedule the deletion of the original blob from the
	// watched data source. nil means no deletion.
	//
	// It is populated via the workflow request.
	RetentionPeriod *time.Duration

	// Whether the top-level directory is meant to be stripped.
	//
	// It is populated via the workflow request.
	StripTopLevelDir bool

	// Key of the blob.
	//
	// It is populated via the workflow request.
	Key string

	// Batch directory that contains the blob.
	//
	// It is populated via the workflow request.
	BatchDir string

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
			CollectionID:     req.CollectionID,
			WatcherName:      req.WatcherName,
			PipelineName:     req.PipelineName,
			RetentionPeriod:  req.RetentionPeriod,
			StripTopLevelDir: req.StripTopLevelDir,
			Key:              req.Key,
			BatchDir:         req.BatchDir,
		}

		// Attributes inferred from the name of the transfer. Populated by parseNameLocalActivity.
		nameInfo nha.NameInfo

		// Collection status. All collections start in queued status.
		status = collection.StatusQueued
	)

	// Persist collection as early as possible.
	{
		activityOpts := withLocalActivityOpts(ctx)
		var err error

		if req.CollectionID == 0 {
			err = workflow.ExecuteLocalActivity(activityOpts, createPackageLocalActivity, w.manager.Logger, w.manager.Collection, &createPackageLocalActivityParams{
				Key:    req.Key,
				Status: status,
			}).Get(activityOpts, &tinfo.CollectionID)
		} else {
			// TODO: investigate better way to reset the collection.
			err = workflow.ExecuteLocalActivity(activityOpts, updatePackageLocalActivity, w.manager.Logger, w.manager.Collection, &updatePackageLocalActivityParams{
				CollectionID: req.CollectionID,
				Key:          req.Key,
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
		dctx, _ := workflow.NewDisconnectedContext(ctx)
		activityOpts := withLocalActivityOpts(dctx)
		_ = workflow.ExecuteLocalActivity(activityOpts, updatePackageLocalActivity, w.manager.Logger, w.manager.Collection, &updatePackageLocalActivityParams{
			CollectionID: tinfo.CollectionID,
			Key:          tinfo.Key,
			PipelineID:   tinfo.PipelineID,
			TransferID:   tinfo.TransferID,
			SIPID:        tinfo.SIPID,
			StoredAt:     tinfo.StoredAt,
			Status:       status,
		}).Get(activityOpts, nil)
	}()

	// Extract details from transfer name.
	{
		activityOpts := withLocalActivityWithoutRetriesOpts(ctx)
		err := workflow.ExecuteLocalActivity(activityOpts, nha_activities.ParseNameLocalActivity, tinfo.Key).Get(activityOpts, &nameInfo)

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
		maxAttempts := 5

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

			// We use this timer to identify transfers that exceeded a deadline.
			// We can't rely on workflow.ErrCanceled because the same context
			// error is seen when the session worker dies.
			timer := NewTimer()

			sessErr = w.SessionHandler(sessCtx, attempt, tinfo, nameInfo, req.ValidationConfig, timer)

			// We want to retry the session if it has been canceled as a result
			// of losing the worker but not otherwise. This scenario seems to be
			// identifiable when we have an error but the root context has not
			// been canceled.
			if sessErr != nil && (errors.Is(sessErr, workflow.ErrSessionFailed) || cadence.IsCanceledError(sessErr)) {
				// Root context canceled, hence workflow canceled.
				if ctx.Err() == workflow.ErrCanceled {
					return nil
				}

				// We're done if the transfer deadline was exceeded.
				if cadence.IsCanceledError(sessErr) && timer.Exceeded() {
					return fmt.Errorf("transfer deadline (%s) exceeded", tinfo.PipelineConfig.TransferDeadline)
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
		if tinfo.WatcherName != "" && status == collection.StatusDone && tinfo.RetentionPeriod != nil {
			err := workflow.NewTimer(ctx, *tinfo.RetentionPeriod).Get(ctx, nil)
			if err != nil {
				logger.Warn("Retention policy timer failed", zap.Error(err))
			} else {
				activityOpts := withActivityOptsForRequest(ctx)
				_ = workflow.ExecuteActivity(activityOpts, activities.DeleteOriginalActivityName, tinfo.WatcherName, tinfo.Key).Get(activityOpts, nil)
			}
		}
	}

	logger.Info(
		"Workflow completed successfully!",
		zap.Uint("collectionID", tinfo.CollectionID),
		zap.String("pipeline", tinfo.PipelineName),
		zap.String("watcher", tinfo.WatcherName),
		zap.String("batchDir", tinfo.BatchDir),
		zap.String("key", tinfo.Key),
		zap.String("status", status.String()),
	)

	return nil
}

// SessionHandler runs activities that belong to the same session.
func (w *ProcessingWorkflow) SessionHandler(sessCtx workflow.Context, attempt int, tinfo *TransferInfo, nameInfo nha.NameInfo, validationConfig validation.Config, timer *Timer) error {
	defer workflow.CompleteSession(sessCtx)

	// Block until pipeline semaphore is acquired. The collection status is set
	// to in-progress as soon as the operation succeeds.
	{
		acquired, err := acquirePipeline(sessCtx, w.manager, tinfo.PipelineName, tinfo.CollectionID)
		if acquired {
			defer func() {
				_ = releasePipeline(sessCtx, w.manager, tinfo.PipelineName)
			}()
		}
		if err != nil {
			return err
		}
	}

	// Download.
	{
		// TODO: even if TempFile is defined, we should confirm that the file is
		// locally available in disk, just in case we're in the context of a
		// session retry where a different working is doing the work. In that
		// case, the activity whould be executed again.
		if tinfo.WatcherName != "" && tinfo.TempFile == "" {
			activityOpts := withActivityOptsForLongLivedRequest(sessCtx)
			err := workflow.ExecuteActivity(activityOpts, activities.DownloadActivityName, tinfo.PipelineName, tinfo.WatcherName, tinfo.Key).Get(activityOpts, &tinfo.TempFile)
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
				Key:              tinfo.Key,
				TempFile:         tinfo.TempFile,
				StripTopLevelDir: tinfo.StripTopLevelDir,
				BatchDir:         tinfo.BatchDir,
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

	{
		// Use our timed context if this transfer has a deadline set.
		duration := tinfo.PipelineConfig.TransferDeadline
		if duration != nil {
			var cancel workflow.CancelFunc
			sessCtx, cancel = timer.WithTimeout(sessCtx, *duration)
			defer cancel()
		}

		err := w.transfer(sessCtx, tinfo)
		if err != nil {
			return err
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

func (w *ProcessingWorkflow) transfer(sessCtx workflow.Context, tinfo *TransferInfo) error {
	// Transfer.
	{
		if tinfo.TransferID == "" {
			transferResponse := activities.TransferActivityResponse{}

			activityOpts := withActivityOptsForRequest(sessCtx)
			err := workflow.ExecuteActivity(activityOpts, activities.TransferActivityName, &activities.TransferActivityParams{
				PipelineName:       tinfo.PipelineName,
				TransferLocationID: tinfo.PipelineConfig.TransferLocationID,
				RelPath:            tinfo.Bundle.RelPath,
				Name:               tinfo.Key,
				ProcessingConfig:   tinfo.PipelineConfig.ProcessingConfig,
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
			Key:          tinfo.Key,
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
			Key:          tinfo.Key,
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

	return nil
}
