// Package workflow contains an experimental workflow for Archivemica transfers.
//
// It's not generalized since it contains client-specific activities. However,
// the long-term goal is to build a system where workflows and activities are
// dynamically set up based on user input.
package workflow

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/artefactual-labs/enduro/internal/amclient"
	"github.com/artefactual-labs/enduro/internal/amclient/bundler"
	"github.com/artefactual-labs/enduro/internal/collection"
	"github.com/artefactual-labs/enduro/internal/pipeline"
	"github.com/artefactual-labs/enduro/internal/watcher"

	"github.com/cenkalti/backoff/v3"
	"github.com/google/uuid"
	"github.com/mholt/archiver"
	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/workflow"
	"go.uber.org/zap"
)

const (
	DownloadActivityName               = "download-activity"
	TransferActivityName               = "transfer-activity"
	PollTransferActivityName           = "poll-transfer-activity"
	PollIngestActivityName             = "poll-ingest-activity"
	UpdateHARIActivityName             = "update-hari-activity"
	UpdateProductionSystemActivityName = "update-production-system-activity"
	CleanUpActivityName                = "clean-up-activity"
	HidePackageActivityName            = "hide-package-activity"
	DeleteOriginalActivityName         = "delete-original-activity"

	processingConfig = "automated"
)

type ProcessingWorkflow struct {
	manager *Manager
}

func NewProcessingWorkflow(m *Manager) *ProcessingWorkflow {
	return &ProcessingWorkflow{manager: m}
}

// TransferInfo is shared state that is passed down to activities. It can be
// useful for hooks that may require quick access to processing state.
// TODO: clean this up, e.g.: it can embed a collection.Collection.
type TransferInfo struct {
	CollectionID     uint               // Enduro internal collection ID.
	Event            *watcher.BlobEvent // Original watcher event.
	Name             string             // Name of the transfer.
	FullPath         string             // Path to the transfer directory in the pipeline.
	RelPath          string             // Path relative to transfer directory in the pipeline.
	ProcessingConfig string             // Archivematica processing configuration.
	AutoApprove      bool               // Archivematica auto-approval setting.
	TransferID       string             // Transfer ID given by Archivematica.
	SIPID            string             // SIP ID given by Archivematica.
	StoredAt         time.Time
	Status           collection.Status

	OriginalID string // Client specific, obtained from name.
	Kind       string // Client specific, obtained from name, e.g. "DPJ-SIP".
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
		CollectionID:     req.CollectionID,
		Event:            req.Event,
		ProcessingConfig: processingConfig,
		AutoApprove:      true,
		OriginalID:       req.Event.NameUUID(),
	}

	// Persist collection as early as possible.
	activityOpts := withLocalActivityOpts(ctx)
	err := workflow.ExecuteLocalActivity(activityOpts, createPackageLocalActivity, w.manager.Collection, tinfo).Get(activityOpts, &tinfo)
	if err != nil {
		return nonRetryableError(fmt.Errorf("Error persisting collection: %w", err))
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
			return nonRetryableError(fmt.Errorf("Error creating session: %w", err))
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
	if tinfo.Status == collection.StatusDone {
		deletionTimer = workflow.NewTimer(ctx, tinfo.Event.RetentionPeriod)
	}

	// Activities that we want to run within the session regardless the
	// result. E.g. receipts, clean-ups, etc...
	// Passing the activity lets the activity determine if the process failed.
	var futures []workflow.Future
	activityOpts = withActivityOptsForRequest(sessCtx)
	futures = append(futures, workflow.ExecuteActivity(activityOpts, UpdateHARIActivityName, tinfo))
	futures = append(futures, workflow.ExecuteActivity(activityOpts, UpdateProductionSystemActivityName, tinfo))
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
	_ = workflow.ExecuteActivity(activityOpts, CleanUpActivityName, tinfo).Get(activityOpts, nil)

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
		zap.String("originalID", tinfo.OriginalID),
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
	err = workflow.ExecuteActivity(activityOpts, DownloadActivityName, tinfo).Get(activityOpts, &tinfo)
	if err != nil {
		return err
	}

	// Transfer.
	//
	// This is our first interaction with Archivematica. The workflow ends here
	// after authentication errors.
	activityOpts = withActivityOptsForRequest(sessCtx)
	err = workflow.ExecuteActivity(activityOpts, TransferActivityName, tinfo).Get(activityOpts, &tinfo)
	if err != nil {
		return err
	}

	activityOpts = withLocalActivityOpts(ctx)
	_ = workflow.ExecuteLocalActivity(activityOpts, updatePackageStatusLocalActivity, w.manager.Collection, tinfo).Get(activityOpts, nil)

	// Poll transfer.
	activityOpts = withActivityOptsForHeartbeatedRequest(sessCtx, time.Minute)
	err = workflow.ExecuteActivity(activityOpts, PollTransferActivityName, tinfo).Get(activityOpts, &tinfo)
	if err != nil {
		return err
	}

	// Poll ingest.
	activityOpts = withActivityOptsForHeartbeatedRequest(sessCtx, time.Minute)
	err = workflow.ExecuteActivity(activityOpts, PollIngestActivityName, tinfo).Get(activityOpts, &tinfo)
	if err != nil {
		return err
	}

	return nil
}

type DownloadActivity struct {
	manager *Manager
}

func NewDownloadActivity(m *Manager) *DownloadActivity {
	return &DownloadActivity{manager: m}
}

// Execute downloads the submitted package.
//
// This implementation is client-specific for now.
// It needs to be cleaned up and generalized.
//
// We're making the following assumptions:
//
// * The blob is a file using the tar or zip archival formats.
// * Expected blob key: `DPJ-SIP-<uuid>.tar`.
// * AVLXML file found at: `DPJ-SIP-<uuid>.tar:/<uuid>/DPJ/journal/<uuid>.xml`.
// * The contents are submitted to Archivematica as-is.
//
// A broader implementation could try to identify the format of the package and
// submit to Archivematica without extracting th econtents.
func (a *DownloadActivity) Execute(ctx context.Context, tinfo *TransferInfo) (*TransferInfo, error) {
	cfg, err := a.manager.Pipelines.Config(tinfo.Event.PipelineName)
	if err != nil {
		return tinfo, nonRetryableError(fmt.Errorf("Error loading pipeline configuration: %v", err))
	}

	var (
		name       string
		kind       string
		path       string
		originalID string
	)

	var isArchived bool
	if _, err := archiver.ByExtension(tinfo.Event.Key); err == nil {
		isArchived = true
	}

	if isArchived {
		//
		// Client-specific, temporary solution.
		//

		// Relevant information is encoded in the name of the blob, e.g.: DPJ-SIP-<uuid>.tar.
		var clientNameRegex = regexp.MustCompile(`^(?P<kind>.*)-(?P<uuid>[a-z0-9]{8}-[a-z0-9]{4}-[1-5][a-z0-9]{3}-[a-z0-9]{4}-[a-z0-9]{12})\.(?P<fileext>.*)`)
		res := clientNameRegex.FindStringSubmatch(tinfo.Event.Key)
		if len(res) < 3 {
			return tinfo, nonRetryableError(fmt.Errorf("Error identifying blob name: %s", tinfo.Event.Key))
		}
		if _, err := uuid.Parse(res[2]); err != nil {
			return tinfo, nonRetryableError(fmt.Errorf("Error identifying UUID in blob name: %v", err))
		}

		// Download the stream in a new temporary file in the local processing directory.
		tmpFile, err := a.manager.Pipelines.TempFile(tinfo.Event.PipelineName, tinfo.Event.Key)
		if err != nil {
			return tinfo, nonRetryableError(fmt.Errorf("Error creating temporary file: %v", err))
		}
		if err := a.manager.Watcher.Download(ctx, tmpFile, tinfo.Event); err != nil {
			return tinfo, nonRetryableError(fmt.Errorf("Error downloading blob: %v", err))
		}

		// Unarchive the file in a new directory inside the transfer location to avoid collisions.
		const tmpDirPrefix = "enduro"
		tmpDir, err := ioutil.TempDir(cfg.TransferDir, tmpDirPrefix)
		if err != nil {
			return tinfo, nonRetryableError(fmt.Errorf("Error creating temporary directory: %s", err))
		}
		_ = os.Chmod(tmpDir, os.FileMode(0o755))
		if err := archiver.Unarchive(tmpFile.Name(), tmpDir); err != nil {
			return tinfo, nonRetryableError(fmt.Errorf("Error unarchiving package: %s", err))
		}

		// Delete the archive. We still have a copy in the watched source.
		_ = os.Remove(tmpFile.Name())

		kind = res[1]                                     // E.g.: DPJ-SIP
		name = fmt.Sprintf("%s-%s", res[1], res[2][0:13]) // E.g.: DPJ-SIP-<uuid[0:13]>
		path = tmpDir                                     // E.g.: /foo/bar/DPJ-SIP-uuid
		originalID = res[2]                               // E.g.: <uuid>
	} else {
		//
		// When we just have a file or an archive with a format that we can't handle.
		//

		b, err := bundler.NewBundlerWithTempDir(cfg.TransferDir)
		if err != nil {
			return tinfo, nonRetryableError(fmt.Errorf("Error bootstrapping bundle: %v", err))
		}

		file, err := b.Create(filepath.Join("objects", tinfo.Event.Key))
		if err != nil {
			return tinfo, nonRetryableError(fmt.Errorf("Error creating file: %v", err))
		}

		if err := a.manager.Watcher.Download(ctx, file, tinfo.Event); err != nil {
			return tinfo, nonRetryableError(fmt.Errorf("Error downloading blob: %v", err))
		}

		if err := b.Bundle(); err != nil {
			return tinfo, nonRetryableError(fmt.Errorf("Error creating transfer bundle: %v", err))
		}

		path = b.FullBaseFsPath()
		name = filepath.Base(path)
	}

	// We must return the relative path which is persisted in the workflow and
	// used by other activities.
	relPath, err := filepath.Rel(cfg.TransferDir, path)
	if err != nil {
		return tinfo, nonRetryableError(fmt.Errorf("Error calculating relative path: %v", err))
	}

	tinfo.Name = name
	tinfo.Kind = kind
	tinfo.OriginalID = originalID
	tinfo.RelPath = relPath
	tinfo.FullPath = path

	return tinfo, nil
}

type TransferActivity struct {
	manager *Manager
}

func NewTransferActivity(m *Manager) *TransferActivity {
	return &TransferActivity{manager: m}
}

func (a *TransferActivity) Execute(ctx context.Context, tinfo *TransferInfo) (*TransferInfo, error) {
	amc, err := a.manager.Pipelines.Client(tinfo.Event.PipelineName)
	if err != nil {
		return nil, err
	}

	config, err := a.manager.Pipelines.Config(tinfo.Event.PipelineName)
	if err != nil {
		return nil, err
	}

	// Transfer path should include the location UUID if defined.
	var path = tinfo.RelPath
	if config.TransferLocationID != "" {
		path = fmt.Sprintf("%s:%s", config.TransferLocationID, path)
	}

	resp, httpResp, err := amc.Package.Create(ctx, &amclient.PackageCreateRequest{
		Name:             tinfo.Name,
		Path:             path,
		ProcessingConfig: tinfo.ProcessingConfig,
		AutoApprove:      &tinfo.AutoApprove,
	})
	if err != nil {
		if httpResp != nil {
			switch {
			case httpResp.StatusCode == http.StatusForbidden:
				return tinfo, nonRetryableError(fmt.Errorf("Authentication error in Archivematica: %v", err))
			}
		}
		return tinfo, err
	}

	tinfo.TransferID = resp.ID

	return tinfo, nil
}

type PollTransferActivity struct {
	manager *Manager
}

func NewPollTransferActivity(m *Manager) *PollTransferActivity {
	return &PollTransferActivity{manager: m}
}

func (a *PollTransferActivity) Execute(ctx context.Context, tinfo *TransferInfo) (*TransferInfo, error) {
	amc, err := a.manager.Pipelines.Client(tinfo.Event.PipelineName)
	if err != nil {
		return tinfo, err
	}

	var sipID string
	var backoffStrategy = backoff.WithContext(backoff.NewConstantBackOff(time.Second*5), ctx)

	err = backoff.RetryNotify(
		func() (err error) {
			ctx, cancel := context.WithTimeout(ctx, time.Second*2)
			defer cancel()

			sipID, err = pipeline.TransferStatus(ctx, amc, tinfo.TransferID)
			if errors.Is(err, pipeline.ErrStatusNonRetryable) {
				return backoff.Permanent(err)
			}

			return err
		},
		backoffStrategy,
		func(err error, duration time.Duration) {
			activity.RecordHeartbeat(ctx, err.Error())
		},
	)

	if err == nil {
		tinfo.SIPID = sipID
	}

	return tinfo, err
}

type PollIngestActivity struct {
	manager *Manager
}

func NewPollIngestActivity(m *Manager) *PollIngestActivity {
	return &PollIngestActivity{manager: m}
}

func (a *PollIngestActivity) Execute(ctx context.Context, tinfo *TransferInfo) (*TransferInfo, error) {
	amc, err := a.manager.Pipelines.Client(tinfo.Event.PipelineName)
	if err != nil {
		return tinfo, err
	}

	var backoffStrategy = backoff.WithContext(backoff.NewConstantBackOff(time.Second*5), ctx)

	err = backoff.RetryNotify(
		func() (err error) {
			ctx, cancel := context.WithTimeout(ctx, time.Second*2)
			defer cancel()

			err = pipeline.IngestStatus(ctx, amc, tinfo.SIPID)
			if errors.Is(err, pipeline.ErrStatusNonRetryable) {
				return backoff.Permanent(err)
			}

			return err
		},
		backoffStrategy,
		func(err error, duration time.Duration) {
			activity.RecordHeartbeat(ctx, err.Error())
		},
	)

	if err == nil {
		tinfo.StoredAt = time.Now().UTC()
	}

	return tinfo, err
}

type CleanUpActivity struct {
	manager *Manager
}

func NewCleanUpActivity(m *Manager) *CleanUpActivity {
	return &CleanUpActivity{manager: m}
}

func (a *CleanUpActivity) Execute(ctx context.Context, tinfo *TransferInfo) error {
	if tinfo.RelPath == "" {
		return nil
	}

	cfg, err := a.manager.Pipelines.Config(tinfo.Event.PipelineName)
	if err != nil {
		return err
	}

	if err := os.RemoveAll(filepath.Join(cfg.TransferDir, tinfo.RelPath)); err != nil {
		return err
	}

	return nil
}

type HidePackageActivity struct {
	manager *Manager
}

func NewHidePackageActivity(m *Manager) *HidePackageActivity {
	return &HidePackageActivity{manager: m}
}

func (a *HidePackageActivity) Execute(ctx context.Context, unitID, unitType, pipelineName string) error {
	amc, err := a.manager.Pipelines.Client(pipelineName)
	if err != nil {
		return nonRetryableError(fmt.Errorf("error looking up pipeline config: %v", err))
	}

	if unitType != "transfer" && unitType != "ingest" {
		return nonRetryableError(fmt.Errorf("unexpected unit type: %s", unitType))
	}

	if unitType == "transfer" {
		resp, _, err := amc.Transfer.Hide(ctx, unitID)
		if err != nil {
			return fmt.Errorf("error hiding transfer: %v", err)
		}
		if !resp.Removed {
			return fmt.Errorf("error hiding transfer: not removed")
		}
	}

	if unitType == "ingest" {
		resp, _, err := amc.Ingest.Hide(ctx, unitID)
		if err != nil {
			return fmt.Errorf("error hiding sip: %v", err)
		}
		if !resp.Removed {
			return fmt.Errorf("error hiding sip: not removed")
		}
	}

	return nil
}

type DeleteOriginalActivity struct {
	manager *Manager
}

func NewDeleteOriginalActivity(m *Manager) *DeleteOriginalActivity {
	return &DeleteOriginalActivity{manager: m}
}

func (a *DeleteOriginalActivity) Execute(ctx context.Context, event *watcher.BlobEvent) error {
	return a.manager.Watcher.Delete(ctx, event)
}

func createPackageLocalActivity(ctx context.Context, colsvc collection.Service, tinfo *TransferInfo) (*TransferInfo, error) {
	info := activity.GetInfo(ctx)
	tinfo.Status = collection.StatusInProgress

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

	return colsvc.UpdateWorkflowStatus(ctx, tinfo.CollectionID, tinfo.Name, info.WorkflowExecution.ID, info.WorkflowExecution.RunID, tinfo.TransferID, tinfo.SIPID, tinfo.Status, tinfo.StoredAt)
}
