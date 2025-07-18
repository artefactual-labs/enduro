// Package workflow contains an experimental workflow for Archivemica transfers.
//
// It's not generalized since it contains client-specific activities. However,
// the long-term goal is to build a system where workflows and activities are
// dynamically set up based on user input.
package workflow

import (
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/artefactual-sdps/temporal-activities/archive"
	"github.com/go-logr/logr"
	temporalsdk_temporal "go.temporal.io/sdk/temporal"
	temporalsdk_workflow "go.temporal.io/sdk/workflow"

	"github.com/artefactual-labs/enduro/internal/collection"
	"github.com/artefactual-labs/enduro/internal/metadata"
	"github.com/artefactual-labs/enduro/internal/nha"
	nha_activities "github.com/artefactual-labs/enduro/internal/nha/activities"
	"github.com/artefactual-labs/enduro/internal/pipeline"
	"github.com/artefactual-labs/enduro/internal/temporal"
	"github.com/artefactual-labs/enduro/internal/validation"
	"github.com/artefactual-labs/enduro/internal/workflow/activities"
	"github.com/artefactual-labs/enduro/internal/workflow/hooks"
)

type ProcessingWorkflow struct {
	hooks            *hooks.Hooks
	colsvc           collection.Service
	pipelineRegistry *pipeline.Registry
	logger           logr.Logger
	config           Config
}

type Config struct {
	ActivityHeartbeatTimeout time.Duration
}

func NewProcessingWorkflow(h *hooks.Hooks, colsvc collection.Service, pipelineRegistry *pipeline.Registry, l logr.Logger, c Config) *ProcessingWorkflow {
	return &ProcessingWorkflow{hooks: h, colsvc: colsvc, pipelineRegistry: pipelineRegistry, logger: l, config: c}
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
	// It is populated via the workflow request. Expect an empty string when
	// the workflow was started by a batch.
	WatcherName string

	// Name of the pipeline to be used for processing.
	//
	// It is populated by this workflow after the list provided by the user or
	// the list of configured pipelines in the system.
	PipelineName string

	// Retention period.
	// Period of time to schedule the deletion of the original blob from the
	// watched data source. nil means no deletion.
	//
	// It is populated via the workflow request.
	RetentionPeriod *time.Duration

	// Directory where the transfer is moved to once processing has completed
	// successfully.
	//
	// It is populated via the workflow request.
	CompletedDir string

	// Whether the top-level directory is meant to be stripped.
	//
	// It is populated via the workflow request.
	StripTopLevelDir bool

	// Whether hidden files are excluded from the transfer
	//
	// It is populated via the workflow request.
	ExcludeHiddenFiles bool

	// Key of the blob.
	//
	// It is populated via the workflow request.
	Key string

	// Whether the blob is a directory (fs watcher)
	//
	// It is populated via the workflow request.
	IsDir bool

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

	// Processing configuration name.
	//
	// It is populated via the workflow request.
	ProcessingConfig string

	// PipelineID is the UUID of the Archivematica pipeline. Extracted from
	// the API response header when the transfer is submitted.
	//
	// It is populated by transferActivity.
	PipelineID string

	// Hooks is the hook config store.
	//
	// It is populated by loadConfigLocalActivity.
	Hooks map[string]map[string]any

	// Information about the bundle (transfer) that we submit to Archivematica.
	// Full path, relative path...
	//
	// It is populated by BundleActivity.
	Bundle activities.BundleActivityResult

	// Archivematica transfer type.
	//
	// It is populated via the workflow request.
	TransferType string

	MetadataConfig metadata.Config
}

func (tinfo TransferInfo) ProcessingConfiguration() string {
	if tinfo.ProcessingConfig != "" {
		return tinfo.ProcessingConfig
	}
	if tinfo.PipelineConfig == nil {
		return ""
	}
	return tinfo.PipelineConfig.ProcessingConfig
}

// ProcessingWorkflow orchestrates all the activities related to the processing
// of a SIP in Archivematica, including is retrieval, creation of transfer,
// etc...
//
// Retrying this workflow would result in a new Archivematica transfer. We  do
// not have a retry policy in place. The user could trigger a new instance via
// the API.
func (w *ProcessingWorkflow) Execute(ctx temporalsdk_workflow.Context, req *collection.ProcessingWorkflowRequest) error {
	var (
		logger = temporalsdk_workflow.GetLogger(ctx)

		tinfo = &TransferInfo{
			CollectionID:       req.CollectionID,
			WatcherName:        req.WatcherName,
			RetentionPeriod:    req.RetentionPeriod,
			CompletedDir:       req.CompletedDir,
			StripTopLevelDir:   req.StripTopLevelDir,
			ExcludeHiddenFiles: req.ExcludeHiddenFiles,
			Key:                req.Key,
			IsDir:              req.IsDir,
			BatchDir:           req.BatchDir,
			ProcessingConfig:   req.ProcessingConfig,
			TransferType:       req.TransferType,
			MetadataConfig:     req.MetadataConfig,
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
			err = temporalsdk_workflow.ExecuteLocalActivity(activityOpts, createPackageLocalActivity, w.logger, w.colsvc, &createPackageLocalActivityParams{
				Key:    req.Key,
				Status: status,
			}).Get(activityOpts, &tinfo.CollectionID)
		} else {
			// TODO: investigate better way to reset the collection.
			err = temporalsdk_workflow.ExecuteLocalActivity(activityOpts, updatePackageLocalActivity, w.logger, w.colsvc, &updatePackageLocalActivityParams{
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
		dctx, _ := temporalsdk_workflow.NewDisconnectedContext(ctx)
		activityOpts := withLocalActivityOpts(dctx)
		_ = temporalsdk_workflow.ExecuteLocalActivity(activityOpts, updatePackageLocalActivity, w.logger, w.colsvc, &updatePackageLocalActivityParams{
			CollectionID: tinfo.CollectionID,
			Key:          tinfo.Key,
			PipelineID:   tinfo.PipelineID,
			TransferID:   tinfo.TransferID,
			SIPID:        tinfo.SIPID,
			StoredAt:     tinfo.StoredAt,
			Status:       status,
		}).Get(activityOpts, nil)
	}()

	// Reject duplicate collection if applicable.
	{
		if req.RejectDuplicates {
			var exists bool
			activityOpts := withLocalActivityOpts(ctx)
			err := temporalsdk_workflow.ExecuteLocalActivity(activityOpts, checkDuplicatePackageLocalActivity, w.logger, w.colsvc, tinfo.CollectionID).Get(activityOpts, &exists)
			if err != nil {
				return fmt.Errorf("error checking duplicate: %v", err)
			}
			if exists {
				return fmt.Errorf("duplicate detected: key: %s", tinfo.Key)
			}
		}
	}

	// Extract details from transfer name.
	{
		activityOpts := withLocalActivityWithoutRetriesOpts(ctx)
		err := temporalsdk_workflow.ExecuteLocalActivity(activityOpts, nha_activities.ParseNameLocalActivity, tinfo.Key).Get(activityOpts, &nameInfo)

		// An error should only stop the workflow if hari/prod activities are enabled.
		hariDisabled, _ := hooks.HookAttrBool(w.hooks.Hooks, "hari", "disabled")
		prodDisabled, _ := hooks.HookAttrBool(w.hooks.Hooks, "prod", "disabled")
		if err != nil && !hariDisabled && !prodDisabled {
			return fmt.Errorf("error parsing transfer name: %v", err)
		}

		if nameInfo.Identifier != "" {
			activityOpts = withLocalActivityOpts(ctx)
			_ = temporalsdk_workflow.ExecuteLocalActivity(activityOpts, setOriginalIDLocalActivity, w.colsvc, tinfo.CollectionID, nameInfo.Identifier).Get(activityOpts, nil)
		}
	}

	tinfo.PipelineName = req.PipelineName

	// Load pipeline configuration and hooks.
	{
		activityOpts := withLocalActivityWithoutRetriesOpts(ctx)
		err := temporalsdk_workflow.ExecuteLocalActivity(activityOpts, loadConfigLocalActivity, w.hooks, w.pipelineRegistry, w.logger, tinfo.PipelineName, tinfo).Get(activityOpts, &tinfo)
		if err != nil {
			return fmt.Errorf("error loading configuration: %v", err)
		}
	}

	// Activities running within a session.
	{
		var sessErr error
		maxAttempts := 5

		for attempt := 1; attempt <= maxAttempts; attempt++ {
			activityOpts := temporalsdk_workflow.WithActivityOptions(ctx, temporalsdk_workflow.ActivityOptions{
				ScheduleToStartTimeout: forever,
				StartToCloseTimeout:    time.Minute,
			})
			sessCtx, err := temporalsdk_workflow.CreateSession(activityOpts, &temporalsdk_workflow.SessionOptions{
				CreationTimeout:  forever,
				ExecutionTimeout: forever,
				HeartbeatTimeout: w.config.ActivityHeartbeatTimeout,
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
			if sessErr != nil && (errors.Is(sessErr, temporalsdk_workflow.ErrSessionFailed) || temporalsdk_temporal.IsCanceledError(sessErr)) {
				// Root context canceled, hence workflow canceled.
				if ctx.Err() == temporalsdk_workflow.ErrCanceled {
					return nil
				}

				// We're done if the transfer deadline was exceeded.
				if temporalsdk_temporal.IsCanceledError(sessErr) && timer.Exceeded() {
					return fmt.Errorf("transfer deadline (%s) exceeded", tinfo.PipelineConfig.TransferDeadline)
				}

				logger.Error("Session failed, will retry shortly (10s)...",
					"rootCtx", ctx.Err(),
					"attemptFailed", attempt,
					"attemptsLeft", maxAttempts-attempt)

				_ = temporalsdk_workflow.Sleep(ctx, time.Second*10)

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
			futures := []temporalsdk_workflow.Future{}
			activityOpts := withActivityOptsForRequest(ctx)
			futures = append(futures, temporalsdk_workflow.ExecuteActivity(activityOpts, activities.HidePackageActivityName, tinfo.TransferID, "transfer", tinfo.PipelineName))
			futures = append(futures, temporalsdk_workflow.ExecuteActivity(activityOpts, activities.HidePackageActivityName, tinfo.SIPID, "ingest", tinfo.PipelineName))
			for _, f := range futures {
				_ = f.Get(activityOpts, nil)
			}
		}
	}

	// Schedule deletion of the original in the watched data source.
	{
		if status == collection.StatusDone {
			if tinfo.RetentionPeriod != nil {
				err := temporalsdk_workflow.NewTimer(ctx, *tinfo.RetentionPeriod).Get(ctx, nil)
				if err != nil {
					logger.Warn("Retention policy timer failed", "error", err)
				} else {
					activityOpts := withActivityOptsForRequest(ctx)
					_ = temporalsdk_workflow.ExecuteActivity(activityOpts, activities.DeleteOriginalActivityName, tinfo.WatcherName, tinfo.BatchDir, tinfo.Key).Get(activityOpts, nil)
				}
			} else if tinfo.CompletedDir != "" {
				activityOpts := withActivityOptsForLocalAction(ctx)
				err := temporalsdk_workflow.ExecuteActivity(activityOpts, activities.DisposeOriginalActivityName, tinfo.WatcherName, tinfo.CompletedDir, tinfo.BatchDir, tinfo.Key).Get(activityOpts, nil)
				if err != nil {
					return err
				}
			}
		}
	}

	logger.Info(
		"Workflow completed successfully!",
		"collectionID", tinfo.CollectionID,
		"pipeline", tinfo.PipelineName,
		"watcher", tinfo.WatcherName,
		"batchDir", tinfo.BatchDir,
		"key", tinfo.Key,
		"status", status.String(),
	)

	return nil
}

// SessionHandler runs activities that belong to the same session.
func (w *ProcessingWorkflow) SessionHandler(sessCtx temporalsdk_workflow.Context, attempt int, tinfo *TransferInfo, nameInfo nha.NameInfo, validationConfig validation.Config, timer *Timer) error {
	defer temporalsdk_workflow.CompleteSession(sessCtx)

	var release releaser

	// Block until pipeline semaphore is acquired. The collection status is set
	// to in-progress as soon as the operation succeeds.
	{
		var acquired bool
		var err error
		acquired, release, err = acquirePipeline(sessCtx, w.colsvc, w.pipelineRegistry, tinfo.PipelineName, tinfo.CollectionID, w.config.ActivityHeartbeatTimeout)
		if acquired {
			defer func() {
				_ = release(sessCtx)
			}()
		}
		if err != nil {
			return err
		}
	}

	// Download.
	{
		if tinfo.WatcherName != "" && !tinfo.IsDir {
			// TODO: even if TempFile is defined, we should confirm that the file is
			// locally available in disk, just in case we're in the context of a
			// session retry where a different worker is doing the work. In that
			// case, the activity would be executed again.
			if tinfo.TempFile == "" {
				activityOpts := withActivityOptsForLongLivedRequest(sessCtx)
				err := temporalsdk_workflow.ExecuteActivity(
					activityOpts,
					activities.DownloadActivityName,
					tinfo.PipelineName,
					tinfo.WatcherName,
					tinfo.Key,
				).Get(activityOpts, &tinfo.TempFile)
				if err != nil {
					return err
				}
			}
		}
	}

	// Both of these values relate to temporary files on Enduro's processing Dir that never get cleaned-up.
	var tempBlob, tempExtracted string
	tempBlob = tinfo.TempFile
	// Extract downloaded archive file contents.
	{
		if tinfo.WatcherName != "" && !tinfo.IsDir {
			activityOpts := withActivityOptsForLocalAction(sessCtx)
			var result archive.ExtractActivityResult
			err := temporalsdk_workflow.ExecuteActivity(
				activityOpts,
				archive.ExtractActivityName,
				&archive.ExtractActivityParams{SourcePath: tinfo.TempFile},
			).Get(activityOpts, &result)
			if err != nil {
				switch err {
				case archive.ErrInvalidArchive:
					// Not an archive file, bundle it as-is (no error).
				default:
					return temporal.NewNonRetryableError(err)
				}
			} else {
				// Continue with the extracted archive contents.
				tinfo.TempFile = result.ExtractPath
				tinfo.StripTopLevelDir = false
				tinfo.IsDir = true
				tempExtracted = result.ExtractPath
			}
		}
	}

	// Bundle.
	{
		if tinfo.Bundle == (activities.BundleActivityResult{}) {
			activityOpts := withActivityOptsForLongLivedRequest(sessCtx)
			err := temporalsdk_workflow.ExecuteActivity(activityOpts, activities.BundleActivityName, &activities.BundleActivityParams{
				TransferDir:        tinfo.PipelineConfig.TransferDir,
				Key:                tinfo.Key,
				IsDir:              tinfo.IsDir,
				TempFile:           tinfo.TempFile,
				StripTopLevelDir:   tinfo.StripTopLevelDir,
				ExcludeHiddenFiles: tinfo.ExcludeHiddenFiles,
				BatchDir:           tinfo.BatchDir,
				Unbag:              tinfo.PipelineConfig.Unbag,
			}).Get(activityOpts, &tinfo.Bundle)
			if err != nil {
				return err
			}
		}
	}

	// Delete local temporary files.
	defer func() {
		// We need disconnected context here because when session gets released the cleanup
		// activities get scheduled and then immediately canceled.
		var filesToRemove []string
		if tinfo.Bundle.FullPathBeforeStrip != "" {
			filesToRemove = append(filesToRemove, tinfo.Bundle.FullPathBeforeStrip)
		}
		if tempBlob != "" {
			filesToRemove = append(filesToRemove, tempBlob)
		}
		if tempExtracted != "" {
			filesToRemove = append(filesToRemove, tempExtracted)
		}
		cleanUpCtx, cancel := temporalsdk_workflow.NewDisconnectedContext(sessCtx)
		defer cancel()
		activityOpts := withActivityOptsForLocalAction(cleanUpCtx)
		if err := temporalsdk_workflow.ExecuteActivity(activityOpts, activities.CleanUpActivityName, &activities.CleanUpActivityParams{
			Paths: filesToRemove,
		}).Get(activityOpts, nil); err != nil {
			w.logger.Error(err, "failed to clean up temporary files", "path", tempExtracted)
		}
	}()

	// Validate transfer.
	{
		if validationConfig.IsEnabled() && tinfo.Bundle != (activities.BundleActivityResult{}) {
			activityOpts := temporalsdk_workflow.WithActivityOptions(sessCtx, temporalsdk_workflow.ActivityOptions{
				ScheduleToStartTimeout: forever,
				StartToCloseTimeout:    time.Minute * 5,
			})
			err := temporalsdk_workflow.ExecuteActivity(activityOpts, activities.ValidateTransferActivityName, &activities.ValidateTransferActivityParams{
				Config: validationConfig,
				Path:   tinfo.Bundle.FullPath,
			}).Get(activityOpts, nil)
			if err != nil {
				return err
			}
		}
	}

	nameMetadata := metadata.TransferName{}
	if tinfo.MetadataConfig.IsEnabled() {
		nameMetadata = metadata.FromTransferName(tinfo.Key, tinfo.IsDir)
	}

	// Populate metadata file with DC identifier.
	{
		if nameMetadata.DCIdentifier != "" {
			activityOpts := temporalsdk_workflow.WithActivityOptions(sessCtx, temporalsdk_workflow.ActivityOptions{
				ScheduleToStartTimeout: forever,
				StartToCloseTimeout:    time.Minute,
			})
			params := activities.PopulateMetadataActivityParams{
				Path:       tinfo.Bundle.FullPath,
				Identifier: nameMetadata.DCIdentifier,
			}
			err := temporalsdk_workflow.ExecuteActivity(activityOpts, activities.PopulateMetadataActivityName, params).Get(activityOpts, nil)
			if err != nil {
				return err
			}
		}
	}

	// Transfer process which is made of multiple activities
	{
		// Use our timed context if this transfer has a deadline set.
		duration := tinfo.PipelineConfig.TransferDeadline
		if duration != nil {
			var cancel temporalsdk_workflow.CancelFunc
			sessCtx, cancel = timer.WithTimeout(sessCtx, *duration)
			defer cancel()
		}

		err := w.transfer(sessCtx, tinfo, nameMetadata)
		if err != nil {
			return err
		}
	}

	// We can release now. Other activities will re-use the session but will not
	// need the pipeline.
	_ = release(sessCtx)

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

	return nil
}

func (w *ProcessingWorkflow) transfer(sessCtx temporalsdk_workflow.Context, tinfo *TransferInfo, nameMetadata metadata.TransferName) error {
	// Transfer.
	{
		if tinfo.TransferID == "" {
			transferResponse := activities.TransferActivityResponse{}
			activityOpts := withActivityOptsForRequest(sessCtx)
			err := temporalsdk_workflow.ExecuteActivity(activityOpts, activities.TransferActivityName, &activities.TransferActivityParams{
				PipelineName:       tinfo.PipelineName,
				TransferLocationID: tinfo.PipelineConfig.TransferLocationID,
				RelPath:            tinfo.Bundle.RelPath,
				Name:               tinfo.Key,
				ProcessingConfig:   tinfo.ProcessingConfiguration(),
				TransferType:       tinfo.TransferType,
				Accession:          nameMetadata.Accession,
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
		_ = temporalsdk_workflow.ExecuteLocalActivity(activityOpts, updatePackageLocalActivity, w.logger, w.colsvc, &updatePackageLocalActivityParams{
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
			activityOpts := withActivityOptsForHeartbeatedRequest(sessCtx, w.config.ActivityHeartbeatTimeout)
			err := temporalsdk_workflow.ExecuteActivity(activityOpts, activities.PollTransferActivityName, &activities.PollTransferActivityParams{
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
		_ = temporalsdk_workflow.ExecuteLocalActivity(activityOpts, updatePackageLocalActivity, w.logger, w.colsvc, &updatePackageLocalActivityParams{
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
			activityOpts := withActivityOptsForHeartbeatedRequest(sessCtx, w.config.ActivityHeartbeatTimeout)
			err := temporalsdk_workflow.ExecuteActivity(activityOpts, activities.PollIngestActivityName, &activities.PollIngestActivityParams{
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

// RandomPipeline will randomly choose the pipeline from the list of names provided. If the
// list is empty then choose one from the list of all configured pipelines.
func RandomPipeline(pipelineNames []string, registry *pipeline.Registry) string {
	names := pipelineNames
	if len(names) < 1 {
		names = registry.Names()
		if len(names) < 1 {
			return ""
		}
	}
	src := rand.NewSource(time.Now().UnixNano())
	rnd := rand.New(src) // #nosec G404 -- not security sensitive.
	return names[rnd.Intn(len(names))]
}
