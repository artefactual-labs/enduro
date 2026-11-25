package workflow

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	bagit "github.com/artefactual-labs/bagit-gython"
	"github.com/artefactual-sdps/temporal-activities/archiveextract"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	temporalsdk_activity "go.temporal.io/sdk/activity"
	temporalsdk_testsuite "go.temporal.io/sdk/testsuite"
	temporalsdk_worker "go.temporal.io/sdk/worker"
	temporalsdk_workflow "go.temporal.io/sdk/workflow"
	"go.uber.org/mock/gomock"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"

	"github.com/artefactual-labs/enduro/internal/collection"
	collectionfake "github.com/artefactual-labs/enduro/internal/collection/fake"
	"github.com/artefactual-labs/enduro/internal/nha"
	nha_activities "github.com/artefactual-labs/enduro/internal/nha/activities"
	"github.com/artefactual-labs/enduro/internal/pipeline"
	"github.com/artefactual-labs/enduro/internal/publisher"
	"github.com/artefactual-labs/enduro/internal/reconciliation"
	"github.com/artefactual-labs/enduro/internal/temporal"
	"github.com/artefactual-labs/enduro/internal/validation"
	"github.com/artefactual-labs/enduro/internal/workflow/activities"
	"github.com/artefactual-labs/enduro/internal/workflow/hooks"
)

type ProcessingWorkflowTestSuite struct {
	suite.Suite
	temporalsdk_testsuite.WorkflowTestSuite

	env *temporalsdk_testsuite.TestWorkflowEnvironment

	hooks *hooks.Hooks

	// Each test registers the workflow with a different name to avoid dups.
	workflow *ProcessingWorkflow
}

func (s *ProcessingWorkflowTestSuite) SetupTest() {
	ctrl := gomock.NewController(s.T())
	s.env = s.NewTestWorkflowEnvironment()
	s.env.SetWorkerOptions(temporalsdk_worker.Options{EnableSessionWorker: true})
	s.env.RegisterActivityWithOptions(func(*activities.ValidateTransferActivityParams) error { return nil },
		temporalsdk_activity.RegisterOptions{Name: activities.ValidateTransferActivityName})
	s.hooks = buildHooks(s.T(), ctrl)
	pipelineRegistry, _ := pipeline.NewPipelineRegistry(logr.Discard(), []pipeline.Config{}, nil, nil)
	s.workflow = NewProcessingWorkflow(s.hooks, collectionfake.NewMockService(ctrl), pipelineRegistry, logr.Discard(), Config{})
}

func (s *ProcessingWorkflowTestSuite) AfterTest(suiteName, testName string) {
	s.env.AssertExpectations(s.T())
}

// Workflow ignores an error in parseName when NHA hooks are disabled.
func (s *ProcessingWorkflowTestSuite) TestParseErrorIsIgnored() {
	s.hooks.Hooks["hari"]["disabled"] = true
	s.hooks.Hooks["prod"]["disabled"] = true

	// Collection is persisted.
	s.env.OnActivity(createPackageLocalActivity, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(uint(12345), nil).Once()

	// parseName is executed with errors.
	s.env.OnActivity(nha_activities.ParseNameLocalActivity, mock.Anything, "key").Return(nil, errors.New("parse error")).Once()

	// loadConfig is executed (workflow continued), returning an error.
	s.env.OnActivity(loadConfigLocalActivity, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("pipeline is unavailable")).Once()

	// Defer updates the package with the error status before returning.
	s.env.OnActivity(updatePackageLocalActivity, mock.Anything, mock.Anything, mock.Anything, &updatePackageLocalActivityParams{
		CollectionID: uint(12345),
		Key:          "key",
		PipelineID:   "",
		TransferID:   "",
		SIPID:        "",
		StoredAt:     time.Time{},
		Status:       collection.StatusError,
	}).Return(nil).Once()

	retentionPeriod := time.Second
	s.env.ExecuteWorkflow(s.workflow.Execute, &collection.ProcessingWorkflowRequest{
		CollectionID:     0,
		WatcherName:      "watcher",
		PipelineName:     "pipeline",
		RetentionPeriod:  &retentionPeriod,
		StripTopLevelDir: true,
		Key:              "key",
	})

	s.True(s.env.IsWorkflowCompleted())
	s.ErrorContains(s.env.GetWorkflowError(), "pipeline is unavailable")
}

// Workflow does not ignore an error in parseName when NHA hooks are enabled.
func (s *ProcessingWorkflowTestSuite) TestParseError() {
	s.hooks.Hooks["hari"]["disabled"] = false
	s.hooks.Hooks["prod"]["disabled"] = false

	// Collection is persisted.
	s.env.OnActivity(createPackageLocalActivity, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(uint(12345), nil).Once()

	// parseName is executed, inject error.
	s.env.OnActivity(nha_activities.ParseNameLocalActivity, mock.Anything, "key").Return(nil, errors.New("parse error")).Once()

	// Defer updates the package with the error status before returning.
	s.env.OnActivity(updatePackageLocalActivity, mock.Anything, mock.Anything, mock.Anything, &updatePackageLocalActivityParams{
		CollectionID: uint(12345),
		Key:          "key",
		PipelineID:   "",
		TransferID:   "",
		SIPID:        "",
		StoredAt:     time.Time{},
		Status:       collection.StatusError,
	}).Return(nil).Once()

	retentionPeriod := time.Second
	s.env.ExecuteWorkflow(s.workflow.Execute, &collection.ProcessingWorkflowRequest{
		CollectionID:     0,
		WatcherName:      "watcher",
		PipelineName:     "pipeline",
		RetentionPeriod:  &retentionPeriod,
		StripTopLevelDir: true,
		Key:              "key",
	})

	s.True(s.env.IsWorkflowCompleted())
	s.ErrorContains(s.env.GetWorkflowError(), "parse error")
}

func TestFinalCollectionStatus(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		status   collection.Status
		canceled bool
		want     collection.Status
	}{
		"done remains done": {
			status: collection.StatusDone,
			want:   collection.StatusDone,
		},
		"abandoned remains abandoned": {
			status: collection.StatusAbandoned,
			want:   collection.StatusAbandoned,
		},
		"non-terminal success path defaults to error": {
			status: collection.StatusQueued,
			want:   collection.StatusError,
		},
		"canceled queued workflow becomes abandoned": {
			status:   collection.StatusQueued,
			canceled: true,
			want:     collection.StatusAbandoned,
		},
		"canceled in-progress workflow becomes abandoned": {
			status:   collection.StatusInProgress,
			canceled: true,
			want:     collection.StatusAbandoned,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := finalCollectionStatus(tc.status, tc.canceled)
			assert.Equal(t, got, tc.want)
		})
	}
}

func (s *ProcessingWorkflowTestSuite) TestReconciliationRetryPreservesExistingState() {
	ctrl := gomock.NewController(s.T())
	pipelineRegistry, err := pipeline.NewPipelineRegistry(logr.Discard(), []pipeline.Config{
		{
			Name: "pipeline",
			ID:   "pipeline-id",
			Recovery: pipeline.RecoveryConfig{
				ReconcileExistingAIP: true,
			},
			StorageServiceURL: "http://user:key@example.com",
		},
	}, nil, nil)
	s.Require().NoError(err)
	s.workflow = NewProcessingWorkflow(s.hooks, collectionfake.NewMockService(ctrl), pipelineRegistry, logr.Discard(), Config{})

	s.env.OnActivity(updatePackageLocalActivity, mock.Anything, mock.Anything, mock.Anything, &updatePackageLocalActivityParams{
		CollectionID: uint(12345),
		Key:          "key",
		PipelineID:   "pipeline-id",
		TransferID:   "transfer-id",
		SIPID:        "aip-id",
		StoredAt:     time.Time{},
		Status:       collection.StatusQueued,

		ProcessingConfig: new("previous-config"),
	}).Return(nil).Once()

	s.env.OnActivity(nha_activities.ParseNameLocalActivity, mock.Anything, "key").Return(&nha.NameInfo{}, nil).Once()
	s.env.OnActivity(loadConfigLocalActivity, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("pipeline is unavailable")).Once()
	s.env.OnActivity(updatePackageLocalActivity, mock.Anything, mock.Anything, mock.Anything, &updatePackageLocalActivityParams{
		CollectionID: uint(12345),
		Key:          "key",
		PipelineID:   "pipeline-id",
		TransferID:   "transfer-id",
		SIPID:        "aip-id",
		StoredAt:     time.Time{},
		Status:       collection.StatusError,

		ProcessingConfig: new("previous-config"),
	}).Return(nil).Once()

	s.env.ExecuteWorkflow(s.workflow.Execute, &collection.ProcessingWorkflowRequest{
		CollectionID:       12345,
		PipelineName:       "pipeline",
		Key:                "key",
		RetryMode:          collection.RetryModeReconcileExistingAIP,
		ExistingTransferID: "transfer-id",
		ExistingAIPID:      "aip-id",
		ExistingPipelineID: "pipeline-id",

		ExistingProcessingConfig: new("previous-config"),
	})

	s.True(s.env.IsWorkflowCompleted())
	s.ErrorContains(s.env.GetWorkflowError(), "pipeline is unavailable")
}

func (s *ProcessingWorkflowTestSuite) TestFullReprocessRetryClearsExistingState() {
	ctrl := gomock.NewController(s.T())
	pipelineRegistry, err := pipeline.NewPipelineRegistry(logr.Discard(), []pipeline.Config{
		{Name: "pipeline", ID: "pipeline-id"},
	}, nil, nil)
	s.Require().NoError(err)
	s.workflow = NewProcessingWorkflow(s.hooks, collectionfake.NewMockService(ctrl), pipelineRegistry, logr.Discard(), Config{})

	s.env.OnActivity(updatePackageLocalActivity, mock.Anything, mock.Anything, mock.Anything, &updatePackageLocalActivityParams{
		CollectionID: uint(12345),
		Key:          "key",
		PipelineID:   "",
		TransferID:   "",
		SIPID:        "",
		StoredAt:     time.Time{},
		Status:       collection.StatusQueued,
	}).Return(nil).Once()
	s.env.OnActivity(updateReconciliationLocalActivity, mock.Anything, mock.Anything, mock.Anything, &updateReconciliationLocalActivityParams{
		CollectionID: uint(12345),
	}).Return(nil).Once()

	s.env.OnActivity(nha_activities.ParseNameLocalActivity, mock.Anything, "key").Return(nil, errors.New("parse error")).Once()
	s.env.OnActivity(updatePackageLocalActivity, mock.Anything, mock.Anything, mock.Anything, &updatePackageLocalActivityParams{
		CollectionID: uint(12345),
		Key:          "key",
		PipelineID:   "",
		TransferID:   "",
		SIPID:        "",
		StoredAt:     time.Time{},
		Status:       collection.StatusError,
	}).Return(nil).Once()

	s.env.ExecuteWorkflow(s.workflow.Execute, &collection.ProcessingWorkflowRequest{
		CollectionID:       12345,
		PipelineName:       "pipeline",
		Key:                "key",
		RetryMode:          collection.RetryModeFullReprocess,
		ExistingTransferID: "transfer-id",
		ExistingAIPID:      "aip-id",
		ExistingPipelineID: "pipeline-id",

		ExistingProcessingConfig: new("previous-config"),
	})

	s.True(s.env.IsWorkflowCompleted())
	s.ErrorContains(s.env.GetWorkflowError(), "parse error")
}

func (s *ProcessingWorkflowTestSuite) TestReconciliationRetryNotFoundFallsBackToFullReprocess() {
	s.hooks.Hooks["hari"]["disabled"] = true
	s.hooks.Hooks["prod"]["disabled"] = true

	ctrl := gomock.NewController(s.T())
	pipelineRegistry, err := pipeline.NewPipelineRegistry(logr.Discard(), []pipeline.Config{
		{Name: "pipeline", ID: "pipeline-id"},
	}, nil, nil)
	s.Require().NoError(err)
	s.workflow = NewProcessingWorkflow(s.hooks, collectionfake.NewMockService(ctrl), pipelineRegistry, logr.Discard(), Config{})
	registerWorkflowActivityStubs(s.env)
	s.env.RegisterActivityWithOptions(func(*activities.ReconcileStorageActivityParams) (*activities.ReconcileStorageActivityResponse, error) {
		return nil, nil
	}, temporalsdk_activity.RegisterOptions{Name: activities.ReconcileStorageActivityName})
	s.env.RegisterActivityWithOptions(func(*activities.BundleActivityParams) (*activities.BundleActivityResult, error) {
		return nil, nil
	}, temporalsdk_activity.RegisterOptions{Name: activities.BundleActivityName})
	s.env.RegisterActivityWithOptions(func(*activities.TransferActivityParams) (*activities.TransferActivityResponse, error) {
		return nil, nil
	}, temporalsdk_activity.RegisterOptions{Name: activities.TransferActivityName})
	s.env.RegisterActivityWithOptions(func(*activities.PollTransferActivityParams) (string, error) {
		return "", nil
	}, temporalsdk_activity.RegisterOptions{Name: activities.PollTransferActivityName})
	s.env.RegisterActivityWithOptions(func(*activities.PollIngestActivityParams) (time.Time, error) {
		return time.Time{}, nil
	}, temporalsdk_activity.RegisterOptions{Name: activities.PollIngestActivityName})
	s.env.RegisterActivityWithOptions(func(*activities.CleanUpActivityParams) error { return nil }, temporalsdk_activity.RegisterOptions{Name: activities.CleanUpActivityName})
	s.env.RegisterActivityWithOptions(func(string, string, string, bool) error { return nil }, temporalsdk_activity.RegisterOptions{Name: activities.HidePackageActivityName})

	storedAt := time.Date(2026, time.March, 17, 9, 0, 0, 0, time.UTC)

	s.env.OnActivity(updatePackageLocalActivity, mock.Anything, mock.Anything, mock.Anything, &updatePackageLocalActivityParams{
		CollectionID: uint(12345),
		Key:          "key",
		PipelineID:   "pipeline-id",
		TransferID:   "transfer-id",
		SIPID:        "aip-id",
		StoredAt:     time.Time{},
		Status:       collection.StatusQueued,

		ProcessingConfig: new("previous-config"),
	}).Return(nil).Once()

	s.env.OnActivity(nha_activities.ParseNameLocalActivity, mock.Anything, "key").Return(&nha.NameInfo{}, nil).Once()
	s.env.OnActivity(loadConfigLocalActivity, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(func(_ context.Context, _ *hooks.Hooks, _ *pipeline.Registry, _ logr.Logger, _ string, tinfo *TransferInfo) (*TransferInfo, error) {
			out := *tinfo
			out.PipelineConfig = &pipeline.Config{
				Name:               "pipeline",
				ID:                 "pipeline-id",
				TransferDir:        "/transfer-dir",
				TransferLocationID: "transfer-location-id",
				ProcessingConfig:   "pipeline-default",
			}
			return &out, nil
		}).Once()

	s.env.OnActivity(activities.AcquirePipelineActivityName, "pipeline").Return(nil).Once()
	s.env.OnActivity(setStatusInProgressLocalActivity, mock.Anything, mock.Anything, uint(12345), mock.Anything).Return(nil).Once()
	s.env.OnActivity(activities.ReconcileStorageActivityName, &activities.ReconcileStorageActivityParams{
		PipelineName: "pipeline",
		AIPID:        "aip-id",
	}).Return(&activities.ReconcileStorageActivityResponse{
		Classification: reconciliation.ClassificationNotFound,
		Status:         reconciliation.StatusPending,
	}, nil).Once()
	s.env.OnActivity(updateReconciliationLocalActivity, mock.Anything, mock.Anything, mock.Anything, mock.MatchedBy(func(params *updateReconciliationLocalActivityParams) bool {
		return params.CollectionID == 12345 &&
			params.AIPStoredAt == nil &&
			params.CheckedAt != nil &&
			params.Status != nil &&
			*params.Status == string(reconciliation.StatusPending)
	})).Return(nil).Once()
	s.env.OnActivity(updatePackageLocalActivity, mock.Anything, mock.Anything, mock.Anything, &updatePackageLocalActivityParams{
		CollectionID: uint(12345),
		Key:          "key",
		PipelineID:   "",
		TransferID:   "",
		SIPID:        "",
		StoredAt:     time.Time{},
		Status:       collection.StatusInProgress,
	}).Return(nil).Once()
	s.env.OnActivity(updateReconciliationLocalActivity, mock.Anything, mock.Anything, mock.Anything, &updateReconciliationLocalActivityParams{
		CollectionID: uint(12345),
	}).Return(nil).Once()
	s.env.OnActivity(activities.BundleActivityName, &activities.BundleActivityParams{
		TransferDir:        "/transfer-dir",
		Key:                "key",
		TempFile:           "",
		StripTopLevelDir:   false,
		ExcludeHiddenFiles: false,
		IsDir:              false,
		BatchDir:           "/batch-dir",
		Unbag:              false,
	}).Return(&activities.BundleActivityResult{
		RelPath:  "key",
		FullPath: "/transfer-dir/key",
	}, nil).Once()
	s.env.OnActivity(activities.ValidateTransferActivityName, &activities.ValidateTransferActivityParams{
		Path:                 "/transfer-dir/key",
		CheckBagCompleteness: true,
	}).Return(nil).Once()
	s.env.OnActivity(activities.TransferActivityName, &activities.TransferActivityParams{
		PipelineName:       "pipeline",
		TransferLocationID: "transfer-location-id",
		RelPath:            "key",
		Name:               "key",
		ProcessingConfig:   "bulk-choice",
		TransferType:       "",
		Accession:          "",
	}).Return(&activities.TransferActivityResponse{
		TransferID: "new-transfer-id",
		PipelineID: "new-pipeline-id",
	}, nil).Once()
	s.env.OnActivity(updatePackageLocalActivity, mock.Anything, mock.Anything, mock.Anything, &updatePackageLocalActivityParams{
		CollectionID: uint(12345),
		Key:          "key",
		PipelineID:   "new-pipeline-id",
		TransferID:   "new-transfer-id",
		SIPID:        "",
		StoredAt:     time.Time{},
		Status:       collection.StatusInProgress,

		ProcessingConfig: new("bulk-choice"),
	}).Return(nil).Once()
	s.env.OnActivity(activities.PollTransferActivityName, &activities.PollTransferActivityParams{
		PipelineName: "pipeline",
		TransferID:   "new-transfer-id",
	}).Return("new-aip-id", nil).Once()
	s.env.OnActivity(updatePackageLocalActivity, mock.Anything, mock.Anything, mock.Anything, &updatePackageLocalActivityParams{
		CollectionID: uint(12345),
		Key:          "key",
		PipelineID:   "new-pipeline-id",
		TransferID:   "new-transfer-id",
		SIPID:        "new-aip-id",
		StoredAt:     time.Time{},
		Status:       collection.StatusInProgress,

		ProcessingConfig: new("bulk-choice"),
	}).Return(nil).Once()
	s.env.OnActivity(activities.PollIngestActivityName, &activities.PollIngestActivityParams{
		PipelineName: "pipeline",
		SIPID:        "new-aip-id",
	}).Return(storedAt, nil).Once()
	s.env.OnActivity(releasePipelineLocalActivity, mock.Anything, mock.Anything, "pipeline").Return(nil).Once()
	s.env.OnActivity(activities.HidePackageActivityName, "new-transfer-id", "transfer", "pipeline", true).Return(nil).Once()
	s.env.OnActivity(activities.HidePackageActivityName, "new-aip-id", "ingest", "pipeline", true).Return(nil).Once()
	s.env.OnActivity(updatePackageLocalActivity, mock.Anything, mock.Anything, mock.Anything, &updatePackageLocalActivityParams{
		CollectionID: uint(12345),
		Key:          "key",
		PipelineID:   "new-pipeline-id",
		TransferID:   "new-transfer-id",
		SIPID:        "new-aip-id",
		StoredAt:     storedAt,
		Status:       collection.StatusDone,

		ProcessingConfig: new("bulk-choice"),
	}).Return(nil).Once()

	s.env.ExecuteWorkflow(s.workflow.Execute, &collection.ProcessingWorkflowRequest{
		CollectionID:       12345,
		PipelineName:       "pipeline",
		Key:                "key",
		BatchDir:           "/batch-dir",
		RetryMode:          collection.RetryModeReconcileExistingAIP,
		ExistingTransferID: "transfer-id",
		ExistingAIPID:      "aip-id",
		ExistingPipelineID: "pipeline-id",

		ExistingProcessingConfig: new("previous-config"),
		ProcessingConfig:         "bulk-choice",
	})

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
}

func (s *ProcessingWorkflowTestSuite) TestReconciliationRetryCompleteDeliversReceipts() {
	s.hooks.Hooks["hari"]["disabled"] = false
	s.hooks.Hooks["prod"]["disabled"] = false

	ctrl := gomock.NewController(s.T())
	pipelineRegistry, err := pipeline.NewPipelineRegistry(logr.Discard(), []pipeline.Config{
		{
			Name: "pipeline",
			ID:   "pipeline-id",
			Recovery: pipeline.RecoveryConfig{
				ReconcileExistingAIP: true,
			},
			StorageServiceURL: "http://user:key@example.com",
		},
	}, nil, nil)
	s.Require().NoError(err)
	s.workflow = NewProcessingWorkflow(s.hooks, collectionfake.NewMockService(ctrl), pipelineRegistry, logr.Discard(), Config{})
	registerWorkflowActivityStubs(s.env)
	s.env.OnActivity(activities.ValidateTransferActivityName, mock.Anything).
		Return(temporal.NewNonRetryableError(errors.New("receipt-only retries must skip validation"))).Maybe()

	completedAt := "2026-03-17T08:00:00Z"
	finalStoredAt := time.Date(2026, time.March, 17, 8, 0, 0, 0, time.UTC)

	s.env.OnActivity(updatePackageLocalActivity, mock.Anything, mock.Anything, mock.Anything, &updatePackageLocalActivityParams{
		CollectionID: uint(12345),
		Key:          "key",
		PipelineID:   "pipeline-id",
		TransferID:   "transfer-id",
		SIPID:        "aip-id",
		StoredAt:     time.Time{},
		Status:       collection.StatusQueued,
	}).Return(nil).Once()
	s.env.OnActivity(nha_activities.ParseNameLocalActivity, mock.Anything, "key").Return(&nha.NameInfo{}, nil).Once()
	s.env.OnActivity(loadConfigLocalActivity, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(func(_ context.Context, _ *hooks.Hooks, _ *pipeline.Registry, _ logr.Logger, _ string, tinfo *TransferInfo) (*TransferInfo, error) {
			out := *tinfo
			out.PipelineConfig = &pipeline.Config{
				Name:        "pipeline",
				ID:          "pipeline-id",
				TransferDir: "/transfer-dir",
				Recovery: pipeline.RecoveryConfig{
					ReconcileExistingAIP: true,
				},
			}
			return &out, nil
		}).Once()
	s.env.OnActivity(activities.AcquirePipelineActivityName, "pipeline").Return(nil).Once()
	s.env.OnActivity(setStatusInProgressLocalActivity, mock.Anything, mock.Anything, uint(12345), mock.Anything).Return(nil).Once()
	s.env.OnActivity(activities.ReconcileStorageActivityName, &activities.ReconcileStorageActivityParams{
		PipelineName: "pipeline",
		AIPID:        "aip-id",
	}).Return(&activities.ReconcileStorageActivityResponse{
		Classification: reconciliation.ClassificationLocalComplete,
		Status:         reconciliation.StatusComplete,
		AIPStoredAt:    &completedAt,
		CompletedAt:    &completedAt,
	}, nil).Once()
	s.env.OnActivity(updateReconciliationLocalActivity, mock.Anything, mock.Anything, mock.Anything, mock.MatchedBy(func(params *updateReconciliationLocalActivityParams) bool {
		return params.CollectionID == 12345 &&
			params.AIPStoredAt != nil &&
			params.CheckedAt != nil &&
			params.Status != nil &&
			*params.Status == string(reconciliation.StatusComplete) &&
			params.Error == nil &&
			params.AIPStoredAt.Equal(finalStoredAt)
	})).Return(nil).Once()
	s.env.OnActivity(releasePipelineLocalActivity, mock.Anything, mock.Anything, "pipeline").Return(nil).Once()
	s.env.OnActivity(activities.BundleActivityName, &activities.BundleActivityParams{
		TransferDir:        "/transfer-dir",
		Key:                "key",
		TempFile:           "",
		StripTopLevelDir:   false,
		ExcludeHiddenFiles: false,
		IsDir:              false,
		BatchDir:           "/batch-dir",
		Unbag:              false,
	}).Return(&activities.BundleActivityResult{
		RelPath:  "key",
		FullPath: "/transfer-dir/key",
	}, nil).Once()
	s.env.OnActivity(nha_activities.UpdateHARIActivityName, &nha_activities.UpdateHARIActivityParams{
		SIPID:        "aip-id",
		StoredAt:     finalStoredAt,
		FullPath:     "/transfer-dir/key",
		PipelineName: "pipeline",
		NameInfo:     nha.NameInfo{},
	}).Return(nil).Once()
	s.env.OnActivity(nha_activities.UpdateProductionSystemActivityName, &nha_activities.UpdateProductionSystemActivityParams{
		StoredAt:     finalStoredAt,
		PipelineName: "pipeline",
		NameInfo:     nha.NameInfo{},
		FullPath:     "/transfer-dir/key",
	}).Return(nil).Once()
	s.env.OnActivity(activities.HidePackageActivityName, "transfer-id", "transfer", "pipeline", true).Return(nil).Once()
	s.env.OnActivity(activities.HidePackageActivityName, "aip-id", "ingest", "pipeline", true).Return(nil).Once()
	s.env.OnActivity(updatePackageLocalActivity, mock.Anything, mock.Anything, mock.Anything, &updatePackageLocalActivityParams{
		CollectionID: uint(12345),
		Key:          "key",
		PipelineID:   "pipeline-id",
		TransferID:   "transfer-id",
		SIPID:        "aip-id",
		StoredAt:     finalStoredAt,
		Status:       collection.StatusDone,
	}).Return(nil).Once()

	s.env.ExecuteWorkflow(s.workflow.Execute, &collection.ProcessingWorkflowRequest{
		CollectionID:       12345,
		PipelineName:       "pipeline",
		Key:                "key",
		BatchDir:           "/batch-dir",
		RetryMode:          collection.RetryModeReconcileExistingAIP,
		ExistingTransferID: "transfer-id",
		ExistingAIPID:      "aip-id",
		ExistingPipelineID: "pipeline-id",
	})

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
	s.env.AssertNotCalled(s.T(), activities.ValidateTransferActivityName, mock.Anything)
}

func (s *ProcessingWorkflowTestSuite) TestReconciliationRetryRequiresExistingAIPID() {
	s.hooks.Hooks["hari"]["disabled"] = true
	s.hooks.Hooks["prod"]["disabled"] = true

	ctrl := gomock.NewController(s.T())
	pipelineRegistry, err := pipeline.NewPipelineRegistry(logr.Discard(), []pipeline.Config{
		{
			Name: "pipeline",
			ID:   "pipeline-id",
			Recovery: pipeline.RecoveryConfig{
				ReconcileExistingAIP: true,
			},
			StorageServiceURL: "http://user:key@example.com",
		},
	}, nil, nil)
	s.Require().NoError(err)
	s.workflow = NewProcessingWorkflow(s.hooks, collectionfake.NewMockService(ctrl), pipelineRegistry, logr.Discard(), Config{})
	registerWorkflowActivityStubs(s.env)

	s.env.OnActivity(updatePackageLocalActivity, mock.Anything, mock.Anything, mock.Anything, &updatePackageLocalActivityParams{
		CollectionID: uint(12345),
		Key:          "key",
		PipelineID:   "pipeline-id",
		TransferID:   "transfer-id",
		SIPID:        "",
		StoredAt:     time.Time{},
		Status:       collection.StatusQueued,
	}).Return(nil).Once()
	s.env.OnActivity(nha_activities.ParseNameLocalActivity, mock.Anything, "key").Return(&nha.NameInfo{}, nil).Once()
	s.env.OnActivity(loadConfigLocalActivity, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(func(_ context.Context, _ *hooks.Hooks, _ *pipeline.Registry, _ logr.Logger, _ string, tinfo *TransferInfo) (*TransferInfo, error) {
			out := *tinfo
			out.PipelineConfig = &pipeline.Config{
				Name: "pipeline",
				ID:   "pipeline-id",
				Recovery: pipeline.RecoveryConfig{
					ReconcileExistingAIP: true,
				},
			}
			return &out, nil
		}).Once()
	s.env.OnActivity(activities.AcquirePipelineActivityName, "pipeline").Return(nil).Once()
	s.env.OnActivity(setStatusInProgressLocalActivity, mock.Anything, mock.Anything, uint(12345), mock.Anything).Return(nil).Once()
	s.env.OnActivity(releasePipelineLocalActivity, mock.Anything, mock.Anything, "pipeline").Return(nil).Once()
	s.env.OnActivity(updatePackageLocalActivity, mock.Anything, mock.Anything, mock.Anything, &updatePackageLocalActivityParams{
		CollectionID: uint(12345),
		Key:          "key",
		PipelineID:   "pipeline-id",
		TransferID:   "transfer-id",
		SIPID:        "",
		StoredAt:     time.Time{},
		Status:       collection.StatusError,
	}).Return(nil).Once()

	s.env.ExecuteWorkflow(s.workflow.Execute, &collection.ProcessingWorkflowRequest{
		CollectionID:       12345,
		PipelineName:       "pipeline",
		Key:                "key",
		RetryMode:          collection.RetryModeReconcileExistingAIP,
		ExistingTransferID: "transfer-id",
		ExistingPipelineID: "pipeline-id",
	})

	s.True(s.env.IsWorkflowCompleted())
	s.ErrorContains(s.env.GetWorkflowError(), "reconciliation retry requires an existing AIP identifier")
	s.True(temporal.NonRetryableError(s.env.GetWorkflowError()))
}

func (s *ProcessingWorkflowTestSuite) TestReconciliationRetryPartialStopsWithoutFullReprocess() {
	s.hooks.Hooks["hari"]["disabled"] = true
	s.hooks.Hooks["prod"]["disabled"] = true

	ctrl := gomock.NewController(s.T())
	pipelineRegistry, err := pipeline.NewPipelineRegistry(logr.Discard(), []pipeline.Config{
		{
			Name: "pipeline",
			ID:   "pipeline-id",
			Recovery: pipeline.RecoveryConfig{
				ReconcileExistingAIP: true,
				RequiredLocations:    []string{"replica-location"},
			},
			StorageServiceURL: "http://user:key@example.com",
		},
	}, nil, nil)
	s.Require().NoError(err)
	s.workflow = NewProcessingWorkflow(s.hooks, collectionfake.NewMockService(ctrl), pipelineRegistry, logr.Discard(), Config{})
	registerWorkflowActivityStubs(s.env)

	s.env.OnActivity(updatePackageLocalActivity, mock.Anything, mock.Anything, mock.Anything, &updatePackageLocalActivityParams{
		CollectionID: uint(12345),
		Key:          "key",
		PipelineID:   "pipeline-id",
		TransferID:   "transfer-id",
		SIPID:        "aip-id",
		StoredAt:     time.Time{},
		Status:       collection.StatusQueued,
	}).Return(nil).Once()
	s.env.OnActivity(nha_activities.ParseNameLocalActivity, mock.Anything, "key").Return(&nha.NameInfo{}, nil).Once()
	s.env.OnActivity(loadConfigLocalActivity, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(func(_ context.Context, _ *hooks.Hooks, _ *pipeline.Registry, _ logr.Logger, _ string, tinfo *TransferInfo) (*TransferInfo, error) {
			out := *tinfo
			out.PipelineConfig = &pipeline.Config{
				Name: "pipeline",
				ID:   "pipeline-id",
				Recovery: pipeline.RecoveryConfig{
					ReconcileExistingAIP: true,
					RequiredLocations:    []string{"replica-location"},
				},
			}
			return &out, nil
		}).Once()
	s.env.OnActivity(activities.AcquirePipelineActivityName, "pipeline").Return(nil).Once()
	s.env.OnActivity(setStatusInProgressLocalActivity, mock.Anything, mock.Anything, uint(12345), mock.Anything).Return(nil).Once()
	s.env.OnActivity(activities.ReconcileStorageActivityName, &activities.ReconcileStorageActivityParams{
		PipelineName: "pipeline",
		AIPID:        "aip-id",
	}).Return(&activities.ReconcileStorageActivityResponse{
		Classification: reconciliation.ClassificationReplicatedPartial,
		Status:         reconciliation.StatusPartial,
	}, nil).Once()
	s.env.OnActivity(updateReconciliationLocalActivity, mock.Anything, mock.Anything, mock.Anything, mock.MatchedBy(func(params *updateReconciliationLocalActivityParams) bool {
		return params.CollectionID == 12345 &&
			params.CheckedAt != nil &&
			params.Status != nil &&
			*params.Status == string(reconciliation.StatusPartial) &&
			params.Error != nil &&
			strings.Contains(*params.Error, string(reconciliation.ClassificationReplicatedPartial))
	})).Return(nil).Once()
	s.env.OnActivity(releasePipelineLocalActivity, mock.Anything, mock.Anything, "pipeline").Return(nil).Once()
	s.env.OnActivity(updatePackageLocalActivity, mock.Anything, mock.Anything, mock.Anything, &updatePackageLocalActivityParams{
		CollectionID: uint(12345),
		Key:          "key",
		PipelineID:   "pipeline-id",
		TransferID:   "transfer-id",
		SIPID:        "aip-id",
		StoredAt:     time.Time{},
		Status:       collection.StatusError,
	}).Return(nil).Once()

	s.env.ExecuteWorkflow(s.workflow.Execute, &collection.ProcessingWorkflowRequest{
		CollectionID:       12345,
		PipelineName:       "pipeline",
		Key:                "key",
		RetryMode:          collection.RetryModeReconcileExistingAIP,
		ExistingTransferID: "transfer-id",
		ExistingAIPID:      "aip-id",
		ExistingPipelineID: "pipeline-id",
	})

	s.True(s.env.IsWorkflowCompleted())
	s.ErrorContains(s.env.GetWorkflowError(), "storage reconciliation incomplete: replicated_partial")
	s.True(temporal.NonRetryableError(s.env.GetWorkflowError()))
}

func (s *ProcessingWorkflowTestSuite) TestReconciliationRetryStorageFailurePersistsUnknownState() {
	s.hooks.Hooks["hari"]["disabled"] = true
	s.hooks.Hooks["prod"]["disabled"] = true

	ctrl := gomock.NewController(s.T())
	pipelineRegistry, err := pipeline.NewPipelineRegistry(logr.Discard(), []pipeline.Config{
		{
			Name: "pipeline",
			ID:   "pipeline-id",
			Recovery: pipeline.RecoveryConfig{
				ReconcileExistingAIP: true,
			},
			StorageServiceURL: "http://user:key@example.com",
		},
	}, nil, nil)
	s.Require().NoError(err)
	s.workflow = NewProcessingWorkflow(s.hooks, collectionfake.NewMockService(ctrl), pipelineRegistry, logr.Discard(), Config{})
	registerWorkflowActivityStubs(s.env)
	reconcileErr := temporal.NewNonRetryableError(errors.New("storage service unavailable"))

	s.env.OnActivity(updatePackageLocalActivity, mock.Anything, mock.Anything, mock.Anything, &updatePackageLocalActivityParams{
		CollectionID: uint(12345),
		Key:          "key",
		PipelineID:   "pipeline-id",
		TransferID:   "transfer-id",
		SIPID:        "aip-id",
		StoredAt:     time.Time{},
		Status:       collection.StatusQueued,
	}).Return(nil).Once()
	s.env.OnActivity(nha_activities.ParseNameLocalActivity, mock.Anything, "key").Return(&nha.NameInfo{}, nil).Once()
	s.env.OnActivity(loadConfigLocalActivity, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(func(_ context.Context, _ *hooks.Hooks, _ *pipeline.Registry, _ logr.Logger, _ string, tinfo *TransferInfo) (*TransferInfo, error) {
			out := *tinfo
			out.PipelineConfig = &pipeline.Config{
				Name: "pipeline",
				ID:   "pipeline-id",
				Recovery: pipeline.RecoveryConfig{
					ReconcileExistingAIP: true,
				},
			}
			return &out, nil
		}).Once()
	s.env.OnActivity(activities.AcquirePipelineActivityName, "pipeline").Return(nil).Once()
	s.env.OnActivity(setStatusInProgressLocalActivity, mock.Anything, mock.Anything, uint(12345), mock.Anything).Return(nil).Once()
	s.env.OnActivity(activities.ReconcileStorageActivityName, &activities.ReconcileStorageActivityParams{
		PipelineName: "pipeline",
		AIPID:        "aip-id",
	}).Return((*activities.ReconcileStorageActivityResponse)(nil), reconcileErr).Once()
	s.env.OnActivity(updateReconciliationLocalActivity, mock.Anything, mock.Anything, mock.Anything, mock.MatchedBy(func(params *updateReconciliationLocalActivityParams) bool {
		return params.CollectionID == 12345 &&
			params.CheckedAt != nil &&
			params.Status != nil &&
			*params.Status == string(reconciliation.StatusUnknown) &&
			params.Error != nil &&
			strings.Contains(*params.Error, "storage service unavailable")
	})).Return(nil).Once()
	s.env.OnActivity(releasePipelineLocalActivity, mock.Anything, mock.Anything, "pipeline").Return(nil).Once()
	s.env.OnActivity(updatePackageLocalActivity, mock.Anything, mock.Anything, mock.Anything, &updatePackageLocalActivityParams{
		CollectionID: uint(12345),
		Key:          "key",
		PipelineID:   "pipeline-id",
		TransferID:   "transfer-id",
		SIPID:        "aip-id",
		StoredAt:     time.Time{},
		Status:       collection.StatusError,
	}).Return(nil).Once()

	s.env.ExecuteWorkflow(s.workflow.Execute, &collection.ProcessingWorkflowRequest{
		CollectionID:       12345,
		PipelineName:       "pipeline",
		Key:                "key",
		RetryMode:          collection.RetryModeReconcileExistingAIP,
		ExistingTransferID: "transfer-id",
		ExistingAIPID:      "aip-id",
		ExistingPipelineID: "pipeline-id",
	})

	s.True(s.env.IsWorkflowCompleted())
	s.ErrorContains(s.env.GetWorkflowError(), "storage service unavailable")
}

func (s *ProcessingWorkflowTestSuite) TestReconciliationRetryIndeterminateStopsWithoutFullReprocess() {
	s.hooks.Hooks["hari"]["disabled"] = true
	s.hooks.Hooks["prod"]["disabled"] = true

	ctrl := gomock.NewController(s.T())
	pipelineRegistry, err := pipeline.NewPipelineRegistry(logr.Discard(), []pipeline.Config{
		{
			Name: "pipeline",
			ID:   "pipeline-id",
			Recovery: pipeline.RecoveryConfig{
				ReconcileExistingAIP: true,
			},
			StorageServiceURL: "http://user:key@example.com",
		},
	}, nil, nil)
	s.Require().NoError(err)
	s.workflow = NewProcessingWorkflow(s.hooks, collectionfake.NewMockService(ctrl), pipelineRegistry, logr.Discard(), Config{})
	registerWorkflowActivityStubs(s.env)

	s.env.OnActivity(updatePackageLocalActivity, mock.Anything, mock.Anything, mock.Anything, &updatePackageLocalActivityParams{
		CollectionID: uint(12345),
		Key:          "key",
		PipelineID:   "pipeline-id",
		TransferID:   "transfer-id",
		SIPID:        "aip-id",
		StoredAt:     time.Time{},
		Status:       collection.StatusQueued,
	}).Return(nil).Once()
	s.env.OnActivity(nha_activities.ParseNameLocalActivity, mock.Anything, "key").Return(&nha.NameInfo{}, nil).Once()
	s.env.OnActivity(loadConfigLocalActivity, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(func(_ context.Context, _ *hooks.Hooks, _ *pipeline.Registry, _ logr.Logger, _ string, tinfo *TransferInfo) (*TransferInfo, error) {
			out := *tinfo
			out.PipelineConfig = &pipeline.Config{
				Name: "pipeline",
				ID:   "pipeline-id",
				Recovery: pipeline.RecoveryConfig{
					ReconcileExistingAIP: true,
				},
			}
			return &out, nil
		}).Once()
	s.env.OnActivity(activities.AcquirePipelineActivityName, "pipeline").Return(nil).Once()
	s.env.OnActivity(setStatusInProgressLocalActivity, mock.Anything, mock.Anything, uint(12345), mock.Anything).Return(nil).Once()
	s.env.OnActivity(activities.ReconcileStorageActivityName, &activities.ReconcileStorageActivityParams{
		PipelineName: "pipeline",
		AIPID:        "aip-id",
	}).Return(&activities.ReconcileStorageActivityResponse{
		Classification: reconciliation.ClassificationIndeterminate,
		Status:         reconciliation.StatusUnknown,
	}, nil).Once()
	s.env.OnActivity(updateReconciliationLocalActivity, mock.Anything, mock.Anything, mock.Anything, mock.MatchedBy(func(params *updateReconciliationLocalActivityParams) bool {
		return params.CollectionID == 12345 &&
			params.CheckedAt != nil &&
			params.Status != nil &&
			*params.Status == string(reconciliation.StatusUnknown) &&
			params.Error != nil &&
			strings.Contains(*params.Error, string(reconciliation.ClassificationIndeterminate))
	})).Return(nil).Once()
	s.env.OnActivity(releasePipelineLocalActivity, mock.Anything, mock.Anything, "pipeline").Return(nil).Once()
	s.env.OnActivity(updatePackageLocalActivity, mock.Anything, mock.Anything, mock.Anything, &updatePackageLocalActivityParams{
		CollectionID: uint(12345),
		Key:          "key",
		PipelineID:   "pipeline-id",
		TransferID:   "transfer-id",
		SIPID:        "aip-id",
		StoredAt:     time.Time{},
		Status:       collection.StatusError,
	}).Return(nil).Once()

	s.env.ExecuteWorkflow(s.workflow.Execute, &collection.ProcessingWorkflowRequest{
		CollectionID:       12345,
		PipelineName:       "pipeline",
		Key:                "key",
		RetryMode:          collection.RetryModeReconcileExistingAIP,
		ExistingTransferID: "transfer-id",
		ExistingAIPID:      "aip-id",
		ExistingPipelineID: "pipeline-id",
	})

	s.True(s.env.IsWorkflowCompleted())
	s.ErrorContains(s.env.GetWorkflowError(), "storage reconciliation incomplete: indeterminate")
	s.True(temporal.NonRetryableError(s.env.GetWorkflowError()))
}

func (s *ProcessingWorkflowTestSuite) TestRecoveryEnabledReconcilesAfterSuccessfulIngest() {
	s.hooks.Hooks["hari"]["disabled"] = true
	s.hooks.Hooks["prod"]["disabled"] = true

	ctrl := gomock.NewController(s.T())
	pipelineRegistry, err := pipeline.NewPipelineRegistry(logr.Discard(), []pipeline.Config{
		{Name: "pipeline", ID: "pipeline-id"},
	}, nil, nil)
	s.Require().NoError(err)
	s.workflow = NewProcessingWorkflow(s.hooks, collectionfake.NewMockService(ctrl), pipelineRegistry, logr.Discard(), Config{})
	registerWorkflowActivityStubs(s.env)

	pollStoredAt := time.Date(2026, time.March, 17, 7, 0, 0, 0, time.UTC)
	aipStoredAt := "2026-03-17T07:00:00Z"
	completedAt := "2026-03-17T08:00:00Z"
	finalStoredAt := time.Date(2026, time.March, 17, 8, 0, 0, 0, time.UTC)

	s.env.OnActivity(updatePackageLocalActivity, mock.Anything, mock.Anything, mock.Anything, &updatePackageLocalActivityParams{
		CollectionID: uint(12345),
		Key:          "key",
		PipelineID:   "",
		TransferID:   "",
		SIPID:        "",
		StoredAt:     time.Time{},
		Status:       collection.StatusQueued,
	}).Return(nil).Once()
	s.env.OnActivity(nha_activities.ParseNameLocalActivity, mock.Anything, "key").Return(&nha.NameInfo{}, nil).Once()
	s.env.OnActivity(loadConfigLocalActivity, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(func(_ context.Context, _ *hooks.Hooks, _ *pipeline.Registry, _ logr.Logger, _ string, tinfo *TransferInfo) (*TransferInfo, error) {
			out := *tinfo
			out.PipelineConfig = &pipeline.Config{
				Name:               "pipeline",
				ID:                 "pipeline-id",
				TransferDir:        "/transfer-dir",
				TransferLocationID: "transfer-location-id",
				ProcessingConfig:   "pipeline-default",
				TransferPublisher: publisher.Config{
					Type: "sftp",
				},
				Recovery: pipeline.RecoveryConfig{
					ReconcileExistingAIP: true,
				},
			}
			return &out, nil
		}).Once()
	s.env.OnActivity(activities.AcquirePipelineActivityName, "pipeline").Return(nil).Once()
	s.env.OnActivity(setStatusInProgressLocalActivity, mock.Anything, mock.Anything, uint(12345), mock.Anything).Return(nil).Once()
	s.env.OnActivity(activities.BundleActivityName, &activities.BundleActivityParams{
		TransferDir:        "/transfer-dir",
		Key:                "key",
		TempFile:           "",
		StripTopLevelDir:   false,
		ExcludeHiddenFiles: false,
		IsDir:              false,
		BatchDir:           "/batch-dir",
		Unbag:              false,
	}).Return(&activities.BundleActivityResult{
		RelPath:  "key",
		FullPath: "/transfer-dir/key",
	}, nil).Once()
	s.env.OnActivity(localTransferPathExistsLocalActivity, "/transfer-dir/key").Return(true, nil).Once()
	s.env.OnActivity(activities.PublishTransferActivityName, &activities.PublishTransferActivityParams{
		PipelineName: "pipeline",
		FullPath:     "/transfer-dir/key",
		RelPath:      "key",
	}).Return(&activities.PublishTransferActivityResult{
		RelPath:    "archivematica/transfers/key",
		RemotePath: "/key",
	}, nil).Once()
	s.env.OnActivity(activities.TransferActivityName, &activities.TransferActivityParams{
		PipelineName:       "pipeline",
		TransferLocationID: "transfer-location-id",
		RelPath:            "archivematica/transfers/key",
		Name:               "key",
		ProcessingConfig:   "pipeline-default",
		TransferType:       "",
		Accession:          "",
	}).Return(&activities.TransferActivityResponse{
		TransferID: "transfer-id",
		PipelineID: "new-pipeline-id",
	}, nil).Once()
	s.env.OnActivity(updatePackageLocalActivity, mock.Anything, mock.Anything, mock.Anything, &updatePackageLocalActivityParams{
		CollectionID: uint(12345),
		Key:          "key",
		PipelineID:   "new-pipeline-id",
		TransferID:   "transfer-id",
		SIPID:        "",
		StoredAt:     time.Time{},
		Status:       collection.StatusInProgress,

		ProcessingConfig: new("pipeline-default"),
	}).Return(nil).Once()
	s.env.OnActivity(activities.PollTransferActivityName, &activities.PollTransferActivityParams{
		PipelineName: "pipeline",
		TransferID:   "transfer-id",
	}).Return("aip-id", nil).Once()
	s.env.OnActivity(updatePackageLocalActivity, mock.Anything, mock.Anything, mock.Anything, &updatePackageLocalActivityParams{
		CollectionID: uint(12345),
		Key:          "key",
		PipelineID:   "new-pipeline-id",
		TransferID:   "transfer-id",
		SIPID:        "aip-id",
		StoredAt:     time.Time{},
		Status:       collection.StatusInProgress,

		ProcessingConfig: new("pipeline-default"),
	}).Return(nil).Once()
	s.env.OnActivity(activities.PollIngestActivityName, &activities.PollIngestActivityParams{
		PipelineName: "pipeline",
		SIPID:        "aip-id",
	}).Return(pollStoredAt, nil).Once()
	s.env.OnActivity(activities.ReconcileStorageActivityName, &activities.ReconcileStorageActivityParams{
		PipelineName: "pipeline",
		AIPID:        "aip-id",
	}).Return(&activities.ReconcileStorageActivityResponse{
		Classification: reconciliation.ClassificationReplicatedComplete,
		Status:         reconciliation.StatusComplete,
		AIPStoredAt:    &aipStoredAt,
		CompletedAt:    &completedAt,
	}, nil).Once()
	s.env.OnActivity(updateReconciliationLocalActivity, mock.Anything, mock.Anything, mock.Anything, mock.MatchedBy(func(params *updateReconciliationLocalActivityParams) bool {
		return params.CollectionID == 12345 &&
			params.AIPStoredAt != nil &&
			params.CheckedAt != nil &&
			params.Status != nil &&
			*params.Status == string(reconciliation.StatusComplete) &&
			params.Error == nil &&
			params.AIPStoredAt.Equal(pollStoredAt)
	})).Return(nil).Once()
	s.env.OnActivity(releasePipelineLocalActivity, mock.Anything, mock.Anything, "pipeline").Return(nil).Once()
	s.env.OnActivity(activities.HidePackageActivityName, "transfer-id", "transfer", "pipeline", false).Return(nil).Once()
	s.env.OnActivity(activities.HidePackageActivityName, "aip-id", "ingest", "pipeline", false).Return(nil).Once()
	s.env.OnActivity(updatePackageLocalActivity, mock.Anything, mock.Anything, mock.Anything, &updatePackageLocalActivityParams{
		CollectionID: uint(12345),
		Key:          "key",
		PipelineID:   "new-pipeline-id",
		TransferID:   "transfer-id",
		SIPID:        "aip-id",
		StoredAt:     finalStoredAt,
		Status:       collection.StatusDone,

		ProcessingConfig: new("pipeline-default"),
	}).Return(nil).Once()

	s.env.ExecuteWorkflow(s.workflow.Execute, &collection.ProcessingWorkflowRequest{
		CollectionID: 12345,
		PipelineName: "pipeline",
		Key:          "key",
		BatchDir:     "/batch-dir",
	})

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
}

func (s *ProcessingWorkflowTestSuite) TestRebuildsBundleWhenPublishingSourceIsMissing() {
	s.env.RegisterActivityWithOptions(func(*activities.BundleActivityParams) (*activities.BundleActivityResult, error) {
		return nil, nil
	}, temporalsdk_activity.RegisterOptions{Name: activities.BundleActivityName})

	s.env.OnActivity(localTransferPathExistsLocalActivity, "/transfer-dir/stale-key").Return(false, nil).Once()
	s.env.OnActivity(activities.BundleActivityName, &activities.BundleActivityParams{
		TransferDir:        "/transfer-dir",
		Key:                "key",
		TempFile:           "",
		StripTopLevelDir:   false,
		ExcludeHiddenFiles: false,
		IsDir:              false,
		BatchDir:           "/batch-dir",
		Unbag:              false,
	}).Return(&activities.BundleActivityResult{
		RelPath:  "rebuilt-key",
		FullPath: "/transfer-dir/rebuilt-key",
	}, nil).Once()

	s.env.ExecuteWorkflow(func(ctx temporalsdk_workflow.Context) error {
		tinfo := &TransferInfo{
			Key:      "key",
			BatchDir: "/batch-dir",
			PipelineConfig: &pipeline.Config{
				TransferDir: "/transfer-dir",
				TransferPublisher: publisher.Config{
					Type: "sftp",
				},
			},
			Bundle: activities.BundleActivityResult{
				RelPath:  "stale-key",
				FullPath: "/transfer-dir/stale-key",
			},
		}

		cleanup, err := s.workflow.rebuildBundleWhenPublishedSourceMissing(ctx, tinfo, &collection.ProcessingWorkflowRequest{
			Key:      "key",
			BatchDir: "/batch-dir",
		}, nil)
		if err != nil {
			return err
		}
		if cleanup == nil {
			return errors.New("expected rebuilt bundle cleanup")
		}
		if tinfo.Bundle.FullPath != "/transfer-dir/rebuilt-key" {
			return fmt.Errorf("unexpected rebuilt bundle path: %s", tinfo.Bundle.FullPath)
		}

		return nil
	})

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
}

func (s *ProcessingWorkflowTestSuite) TestRecoveryEnabledRetriesIndeterminateAfterSuccessfulIngest() {
	s.hooks.Hooks["hari"]["disabled"] = true
	s.hooks.Hooks["prod"]["disabled"] = true

	ctrl := gomock.NewController(s.T())
	pipelineRegistry, err := pipeline.NewPipelineRegistry(logr.Discard(), []pipeline.Config{
		{Name: "pipeline", ID: "pipeline-id"},
	}, nil, nil)
	s.Require().NoError(err)
	s.workflow = NewProcessingWorkflow(s.hooks, collectionfake.NewMockService(ctrl), pipelineRegistry, logr.Discard(), Config{})
	registerWorkflowActivityStubs(s.env)

	pollStoredAt := time.Date(2026, time.March, 17, 7, 0, 0, 0, time.UTC)
	aipStoredAt := "2026-03-17T07:00:00Z"
	completedAt := "2026-03-17T07:00:00Z"
	finalStoredAt := time.Date(2026, time.March, 17, 7, 0, 0, 0, time.UTC)

	s.env.OnActivity(updatePackageLocalActivity, mock.Anything, mock.Anything, mock.Anything, &updatePackageLocalActivityParams{
		CollectionID: uint(12345),
		Key:          "key",
		PipelineID:   "",
		TransferID:   "",
		SIPID:        "",
		StoredAt:     time.Time{},
		Status:       collection.StatusQueued,
	}).Return(nil).Once()
	s.env.OnActivity(nha_activities.ParseNameLocalActivity, mock.Anything, "key").Return(&nha.NameInfo{}, nil).Once()
	s.env.OnActivity(loadConfigLocalActivity, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(func(_ context.Context, _ *hooks.Hooks, _ *pipeline.Registry, _ logr.Logger, _ string, tinfo *TransferInfo) (*TransferInfo, error) {
			out := *tinfo
			out.PipelineConfig = &pipeline.Config{
				Name:               "pipeline",
				ID:                 "pipeline-id",
				TransferDir:        "/transfer-dir",
				TransferLocationID: "transfer-location-id",
				Recovery: pipeline.RecoveryConfig{
					ReconcileExistingAIP: true,
				},
			}
			return &out, nil
		}).Once()
	s.env.OnActivity(activities.AcquirePipelineActivityName, "pipeline").Return(nil).Once()
	s.env.OnActivity(setStatusInProgressLocalActivity, mock.Anything, mock.Anything, uint(12345), mock.Anything).Return(nil).Once()
	s.env.OnActivity(activities.BundleActivityName, &activities.BundleActivityParams{
		TransferDir:        "/transfer-dir",
		Key:                "key",
		TempFile:           "",
		StripTopLevelDir:   false,
		ExcludeHiddenFiles: false,
		IsDir:              false,
		BatchDir:           "/batch-dir",
		Unbag:              false,
	}).Return(&activities.BundleActivityResult{
		RelPath:  "key",
		FullPath: "/transfer-dir/key",
	}, nil).Once()
	s.env.OnActivity(activities.TransferActivityName, &activities.TransferActivityParams{
		PipelineName:       "pipeline",
		TransferLocationID: "transfer-location-id",
		RelPath:            "key",
		Name:               "key",
		ProcessingConfig:   "",
		TransferType:       "",
		Accession:          "",
	}).Return(&activities.TransferActivityResponse{
		TransferID: "transfer-id",
		PipelineID: "new-pipeline-id",
	}, nil).Once()
	s.env.OnActivity(updatePackageLocalActivity, mock.Anything, mock.Anything, mock.Anything, &updatePackageLocalActivityParams{
		CollectionID: uint(12345),
		Key:          "key",
		PipelineID:   "new-pipeline-id",
		TransferID:   "transfer-id",
		SIPID:        "",
		StoredAt:     time.Time{},
		Status:       collection.StatusInProgress,

		ProcessingConfig: new(""),
	}).Return(nil).Once()
	s.env.OnActivity(activities.PollTransferActivityName, &activities.PollTransferActivityParams{
		PipelineName: "pipeline",
		TransferID:   "transfer-id",
	}).Return("aip-id", nil).Once()
	s.env.OnActivity(updatePackageLocalActivity, mock.Anything, mock.Anything, mock.Anything, &updatePackageLocalActivityParams{
		CollectionID: uint(12345),
		Key:          "key",
		PipelineID:   "new-pipeline-id",
		TransferID:   "transfer-id",
		SIPID:        "aip-id",
		StoredAt:     time.Time{},
		Status:       collection.StatusInProgress,

		ProcessingConfig: new(""),
	}).Return(nil).Once()
	s.env.OnActivity(activities.PollIngestActivityName, &activities.PollIngestActivityParams{
		PipelineName: "pipeline",
		SIPID:        "aip-id",
	}).Return(pollStoredAt, nil).Once()

	// First reconciliation returns indeterminate (SS hasn't finished storing yet).
	s.env.OnActivity(activities.ReconcileStorageActivityName, &activities.ReconcileStorageActivityParams{
		PipelineName: "pipeline",
		AIPID:        "aip-id",
	}).Return(&activities.ReconcileStorageActivityResponse{
		Classification: reconciliation.ClassificationIndeterminate,
		Status:         reconciliation.StatusUnknown,
		PrimaryExists:  true,
	}, nil).Once()
	s.env.OnActivity(updateReconciliationLocalActivity, mock.Anything, mock.Anything, mock.Anything, mock.MatchedBy(func(params *updateReconciliationLocalActivityParams) bool {
		return params.CollectionID == 12345 &&
			params.CheckedAt != nil &&
			params.Status != nil &&
			*params.Status == string(reconciliation.StatusUnknown) &&
			params.Error != nil &&
			strings.Contains(*params.Error, string(reconciliation.ClassificationIndeterminate))
	})).Return(nil).Once()

	// Second reconciliation returns local_complete (SS has finished).
	s.env.OnActivity(activities.ReconcileStorageActivityName, &activities.ReconcileStorageActivityParams{
		PipelineName: "pipeline",
		AIPID:        "aip-id",
	}).Return(&activities.ReconcileStorageActivityResponse{
		Classification: reconciliation.ClassificationLocalComplete,
		Status:         reconciliation.StatusComplete,
		PrimaryExists:  true,
		AIPStoredAt:    &aipStoredAt,
		CompletedAt:    &completedAt,
	}, nil).Once()
	s.env.OnActivity(updateReconciliationLocalActivity, mock.Anything, mock.Anything, mock.Anything, mock.MatchedBy(func(params *updateReconciliationLocalActivityParams) bool {
		return params.CollectionID == 12345 &&
			params.AIPStoredAt != nil &&
			params.CheckedAt != nil &&
			params.Status != nil &&
			*params.Status == string(reconciliation.StatusComplete) &&
			params.Error == nil
	})).Return(nil).Once()

	s.env.OnActivity(releasePipelineLocalActivity, mock.Anything, mock.Anything, "pipeline").Return(nil).Once()
	s.env.OnActivity(activities.HidePackageActivityName, "transfer-id", "transfer", "pipeline", false).Return(nil).Once()
	s.env.OnActivity(activities.HidePackageActivityName, "aip-id", "ingest", "pipeline", false).Return(nil).Once()
	s.env.OnActivity(updatePackageLocalActivity, mock.Anything, mock.Anything, mock.Anything, &updatePackageLocalActivityParams{
		CollectionID: uint(12345),
		Key:          "key",
		PipelineID:   "new-pipeline-id",
		TransferID:   "transfer-id",
		SIPID:        "aip-id",
		StoredAt:     finalStoredAt,
		Status:       collection.StatusDone,

		ProcessingConfig: new(""),
	}).Return(nil).Once()

	s.env.ExecuteWorkflow(s.workflow.Execute, &collection.ProcessingWorkflowRequest{
		CollectionID: 12345,
		PipelineName: "pipeline",
		Key:          "key",
		BatchDir:     "/batch-dir",
	})

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
}

func (s *ProcessingWorkflowTestSuite) TestRecoveryEnabledRetriesNotFoundAfterSuccessfulIngest() {
	s.hooks.Hooks["hari"]["disabled"] = true
	s.hooks.Hooks["prod"]["disabled"] = true

	ctrl := gomock.NewController(s.T())
	pipelineRegistry, err := pipeline.NewPipelineRegistry(logr.Discard(), []pipeline.Config{
		{Name: "pipeline", ID: "pipeline-id"},
	}, nil, nil)
	s.Require().NoError(err)
	s.workflow = NewProcessingWorkflow(s.hooks, collectionfake.NewMockService(ctrl), pipelineRegistry, logr.Discard(), Config{})
	registerWorkflowActivityStubs(s.env)

	pollStoredAt := time.Date(2026, time.March, 17, 7, 0, 0, 0, time.UTC)

	s.env.OnActivity(updatePackageLocalActivity, mock.Anything, mock.Anything, mock.Anything, &updatePackageLocalActivityParams{
		CollectionID: uint(12345),
		Key:          "key",
		PipelineID:   "",
		TransferID:   "",
		SIPID:        "",
		StoredAt:     time.Time{},
		Status:       collection.StatusQueued,
	}).Return(nil).Once()
	s.env.OnActivity(nha_activities.ParseNameLocalActivity, mock.Anything, "key").Return(&nha.NameInfo{}, nil).Once()
	s.env.OnActivity(loadConfigLocalActivity, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(func(_ context.Context, _ *hooks.Hooks, _ *pipeline.Registry, _ logr.Logger, _ string, tinfo *TransferInfo) (*TransferInfo, error) {
			out := *tinfo
			out.PipelineConfig = &pipeline.Config{
				Name:               "pipeline",
				ID:                 "pipeline-id",
				TransferDir:        "/transfer-dir",
				TransferLocationID: "transfer-location-id",
				Recovery: pipeline.RecoveryConfig{
					ReconcileExistingAIP: true,
				},
			}
			return &out, nil
		}).Once()
	s.env.OnActivity(activities.AcquirePipelineActivityName, "pipeline").Return(nil).Once()
	s.env.OnActivity(setStatusInProgressLocalActivity, mock.Anything, mock.Anything, uint(12345), mock.Anything).Return(nil).Once()
	s.env.OnActivity(activities.BundleActivityName, &activities.BundleActivityParams{
		TransferDir:        "/transfer-dir",
		Key:                "key",
		TempFile:           "",
		StripTopLevelDir:   false,
		ExcludeHiddenFiles: false,
		IsDir:              false,
		BatchDir:           "/batch-dir",
		Unbag:              false,
	}).Return(&activities.BundleActivityResult{
		RelPath:  "key",
		FullPath: "/transfer-dir/key",
	}, nil).Once()
	s.env.OnActivity(activities.TransferActivityName, &activities.TransferActivityParams{
		PipelineName:       "pipeline",
		TransferLocationID: "transfer-location-id",
		RelPath:            "key",
		Name:               "key",
		ProcessingConfig:   "",
		TransferType:       "",
		Accession:          "",
	}).Return(&activities.TransferActivityResponse{
		TransferID: "transfer-id",
		PipelineID: "new-pipeline-id",
	}, nil).Once()
	s.env.OnActivity(updatePackageLocalActivity, mock.Anything, mock.Anything, mock.Anything, &updatePackageLocalActivityParams{
		CollectionID: uint(12345),
		Key:          "key",
		PipelineID:   "new-pipeline-id",
		TransferID:   "transfer-id",
		SIPID:        "",
		StoredAt:     time.Time{},
		Status:       collection.StatusInProgress,

		ProcessingConfig: new(""),
	}).Return(nil).Once()
	s.env.OnActivity(activities.PollTransferActivityName, &activities.PollTransferActivityParams{
		PipelineName: "pipeline",
		TransferID:   "transfer-id",
	}).Return("aip-id", nil).Once()
	s.env.OnActivity(updatePackageLocalActivity, mock.Anything, mock.Anything, mock.Anything, &updatePackageLocalActivityParams{
		CollectionID: uint(12345),
		Key:          "key",
		PipelineID:   "new-pipeline-id",
		TransferID:   "transfer-id",
		SIPID:        "aip-id",
		StoredAt:     time.Time{},
		Status:       collection.StatusInProgress,

		ProcessingConfig: new(""),
	}).Return(nil).Once()
	s.env.OnActivity(activities.PollIngestActivityName, &activities.PollIngestActivityParams{
		PipelineName: "pipeline",
		SIPID:        "aip-id",
	}).Return(pollStoredAt, nil).Once()
	s.env.OnActivity(activities.ReconcileStorageActivityName, &activities.ReconcileStorageActivityParams{
		PipelineName: "pipeline",
		AIPID:        "aip-id",
	}).Return(&activities.ReconcileStorageActivityResponse{
		Classification: reconciliation.ClassificationNotFound,
		Status:         reconciliation.StatusPending,
	}, nil).Once()
	s.env.OnActivity(updateReconciliationLocalActivity, mock.Anything, mock.Anything, mock.Anything, mock.MatchedBy(func(params *updateReconciliationLocalActivityParams) bool {
		return params.CollectionID == 12345 &&
			params.AIPStoredAt == nil &&
			params.CheckedAt != nil &&
			params.Status != nil &&
			*params.Status == string(reconciliation.StatusPending) &&
			params.Error != nil &&
			strings.Contains(*params.Error, string(reconciliation.ClassificationNotFound))
	})).Return(nil).Once()
	aipStoredAt := "2026-03-17T07:00:00Z"
	completedAt := "2026-03-17T08:00:00Z"
	finalStoredAt := time.Date(2026, time.March, 17, 8, 0, 0, 0, time.UTC)
	s.env.OnActivity(activities.ReconcileStorageActivityName, &activities.ReconcileStorageActivityParams{
		PipelineName: "pipeline",
		AIPID:        "aip-id",
	}).Return(&activities.ReconcileStorageActivityResponse{
		Classification: reconciliation.ClassificationReplicatedComplete,
		Status:         reconciliation.StatusComplete,
		AIPStoredAt:    &aipStoredAt,
		CompletedAt:    &completedAt,
	}, nil).Once()
	s.env.OnActivity(updateReconciliationLocalActivity, mock.Anything, mock.Anything, mock.Anything, mock.MatchedBy(func(params *updateReconciliationLocalActivityParams) bool {
		return params.CollectionID == 12345 &&
			params.AIPStoredAt != nil &&
			params.CheckedAt != nil &&
			params.Status != nil &&
			*params.Status == string(reconciliation.StatusComplete) &&
			params.Error == nil &&
			params.AIPStoredAt.Equal(pollStoredAt)
	})).Return(nil).Once()
	s.env.OnActivity(releasePipelineLocalActivity, mock.Anything, mock.Anything, "pipeline").Return(nil).Once()
	s.env.OnActivity(activities.HidePackageActivityName, "transfer-id", "transfer", "pipeline", false).Return(nil).Once()
	s.env.OnActivity(activities.HidePackageActivityName, "aip-id", "ingest", "pipeline", false).Return(nil).Once()
	s.env.OnActivity(updatePackageLocalActivity, mock.Anything, mock.Anything, mock.Anything, &updatePackageLocalActivityParams{
		CollectionID: uint(12345),
		Key:          "key",
		PipelineID:   "new-pipeline-id",
		TransferID:   "transfer-id",
		SIPID:        "aip-id",
		StoredAt:     finalStoredAt,
		Status:       collection.StatusDone,

		ProcessingConfig: new(""),
	}).Return(nil).Once()

	s.env.ExecuteWorkflow(s.workflow.Execute, &collection.ProcessingWorkflowRequest{
		CollectionID: 12345,
		PipelineName: "pipeline",
		Key:          "key",
		BatchDir:     "/batch-dir",
	})

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
}

func (s *ProcessingWorkflowTestSuite) TestRecoveryEnabledFailsWhenNotFoundPersistsPastRetryWindow() {
	s.hooks.Hooks["hari"]["disabled"] = true
	s.hooks.Hooks["prod"]["disabled"] = true

	ctrl := gomock.NewController(s.T())
	pipelineRegistry, err := pipeline.NewPipelineRegistry(logr.Discard(), []pipeline.Config{
		{Name: "pipeline", ID: "pipeline-id"},
	}, nil, nil)
	s.Require().NoError(err)
	s.workflow = NewProcessingWorkflow(s.hooks, collectionfake.NewMockService(ctrl), pipelineRegistry, logr.Discard(), Config{})
	registerWorkflowActivityStubs(s.env)

	pollStoredAt := time.Date(2026, time.March, 17, 7, 0, 0, 0, time.UTC)

	s.env.OnActivity(updatePackageLocalActivity, mock.Anything, mock.Anything, mock.Anything, &updatePackageLocalActivityParams{
		CollectionID: uint(12345),
		Key:          "key",
		PipelineID:   "",
		TransferID:   "",
		SIPID:        "",
		StoredAt:     time.Time{},
		Status:       collection.StatusQueued,
	}).Return(nil).Once()
	s.env.OnActivity(nha_activities.ParseNameLocalActivity, mock.Anything, "key").Return(&nha.NameInfo{}, nil).Once()
	s.env.OnActivity(loadConfigLocalActivity, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(func(_ context.Context, _ *hooks.Hooks, _ *pipeline.Registry, _ logr.Logger, _ string, tinfo *TransferInfo) (*TransferInfo, error) {
			out := *tinfo
			out.PipelineConfig = &pipeline.Config{
				Name:               "pipeline",
				ID:                 "pipeline-id",
				TransferDir:        "/transfer-dir",
				TransferLocationID: "transfer-location-id",
				Recovery: pipeline.RecoveryConfig{
					ReconcileExistingAIP: true,
				},
			}
			return &out, nil
		}).Once()
	s.env.OnActivity(activities.AcquirePipelineActivityName, "pipeline").Return(nil).Once()
	s.env.OnActivity(setStatusInProgressLocalActivity, mock.Anything, mock.Anything, uint(12345), mock.Anything).Return(nil).Once()
	s.env.OnActivity(activities.BundleActivityName, &activities.BundleActivityParams{
		TransferDir:        "/transfer-dir",
		Key:                "key",
		TempFile:           "",
		StripTopLevelDir:   false,
		ExcludeHiddenFiles: false,
		IsDir:              false,
		BatchDir:           "/batch-dir",
		Unbag:              false,
	}).Return(&activities.BundleActivityResult{
		RelPath:  "key",
		FullPath: "/transfer-dir/key",
	}, nil).Once()
	s.env.OnActivity(activities.TransferActivityName, &activities.TransferActivityParams{
		PipelineName:       "pipeline",
		TransferLocationID: "transfer-location-id",
		RelPath:            "key",
		Name:               "key",
		ProcessingConfig:   "",
		TransferType:       "",
		Accession:          "",
	}).Return(&activities.TransferActivityResponse{
		TransferID: "transfer-id",
		PipelineID: "new-pipeline-id",
	}, nil).Once()
	s.env.OnActivity(updatePackageLocalActivity, mock.Anything, mock.Anything, mock.Anything, &updatePackageLocalActivityParams{
		CollectionID: uint(12345),
		Key:          "key",
		PipelineID:   "new-pipeline-id",
		TransferID:   "transfer-id",
		SIPID:        "",
		StoredAt:     time.Time{},
		Status:       collection.StatusInProgress,

		ProcessingConfig: new(""),
	}).Return(nil).Once()
	s.env.OnActivity(activities.PollTransferActivityName, &activities.PollTransferActivityParams{
		PipelineName: "pipeline",
		TransferID:   "transfer-id",
	}).Return("aip-id", nil).Once()
	s.env.OnActivity(updatePackageLocalActivity, mock.Anything, mock.Anything, mock.Anything, &updatePackageLocalActivityParams{
		CollectionID: uint(12345),
		Key:          "key",
		PipelineID:   "new-pipeline-id",
		TransferID:   "transfer-id",
		SIPID:        "aip-id",
		StoredAt:     time.Time{},
		Status:       collection.StatusInProgress,

		ProcessingConfig: new(""),
	}).Return(nil).Once()
	s.env.OnActivity(activities.PollIngestActivityName, &activities.PollIngestActivityParams{
		PipelineName: "pipeline",
		SIPID:        "aip-id",
	}).Return(pollStoredAt, nil).Once()

	attempts := int(postIngestReconciliationRetryWindow/postIngestReconciliationRetryInterval) + 1
	for range attempts {
		s.env.OnActivity(activities.ReconcileStorageActivityName, &activities.ReconcileStorageActivityParams{
			PipelineName: "pipeline",
			AIPID:        "aip-id",
		}).Return(&activities.ReconcileStorageActivityResponse{
			Classification: reconciliation.ClassificationNotFound,
			Status:         reconciliation.StatusPending,
		}, nil).Once()
		s.env.OnActivity(updateReconciliationLocalActivity, mock.Anything, mock.Anything, mock.Anything, mock.MatchedBy(func(params *updateReconciliationLocalActivityParams) bool {
			return params.CollectionID == 12345 &&
				params.AIPStoredAt == nil &&
				params.CheckedAt != nil &&
				params.Status != nil &&
				*params.Status == string(reconciliation.StatusPending) &&
				params.Error != nil &&
				strings.Contains(*params.Error, string(reconciliation.ClassificationNotFound))
		})).Return(nil).Once()
	}

	s.env.OnActivity(releasePipelineLocalActivity, mock.Anything, mock.Anything, "pipeline").Return(nil).Once()
	s.env.OnActivity(updatePackageLocalActivity, mock.Anything, mock.Anything, mock.Anything, &updatePackageLocalActivityParams{
		CollectionID: uint(12345),
		Key:          "key",
		PipelineID:   "new-pipeline-id",
		TransferID:   "transfer-id",
		SIPID:        "aip-id",
		StoredAt:     time.Time{},
		Status:       collection.StatusError,

		ProcessingConfig: new(""),
	}).Return(nil).Once()

	s.env.ExecuteWorkflow(s.workflow.Execute, &collection.ProcessingWorkflowRequest{
		CollectionID: 12345,
		PipelineName: "pipeline",
		Key:          "key",
		BatchDir:     "/batch-dir",
	})

	s.True(s.env.IsWorkflowCompleted())
	s.ErrorContains(s.env.GetWorkflowError(), "storage reconciliation returned not_found")
	s.True(temporal.NonRetryableError(s.env.GetWorkflowError()))
}

func (s *ProcessingWorkflowTestSuite) TestRecoveryEnabledPromotesIngestFailureWhenReconciliationCompletes() {
	s.hooks.Hooks["hari"]["disabled"] = true
	s.hooks.Hooks["prod"]["disabled"] = true

	ctrl := gomock.NewController(s.T())
	pipelineRegistry, err := pipeline.NewPipelineRegistry(logr.Discard(), []pipeline.Config{
		{Name: "pipeline", ID: "pipeline-id"},
	}, nil, nil)
	s.Require().NoError(err)
	s.workflow = NewProcessingWorkflow(s.hooks, collectionfake.NewMockService(ctrl), pipelineRegistry, logr.Discard(), Config{})
	registerWorkflowActivityStubs(s.env)

	pollErr := errors.New("ingest poll failed")
	aipStoredAt := "2026-03-17T07:00:00Z"
	completedAt := "2026-03-17T08:00:00Z"
	finalStoredAt := time.Date(2026, time.March, 17, 8, 0, 0, 0, time.UTC)

	s.env.OnActivity(updatePackageLocalActivity, mock.Anything, mock.Anything, mock.Anything, &updatePackageLocalActivityParams{
		CollectionID: uint(12345),
		Key:          "key",
		PipelineID:   "",
		TransferID:   "",
		SIPID:        "",
		StoredAt:     time.Time{},
		Status:       collection.StatusQueued,
	}).Return(nil).Once()
	s.env.OnActivity(nha_activities.ParseNameLocalActivity, mock.Anything, "key").Return(&nha.NameInfo{}, nil).Once()
	s.env.OnActivity(loadConfigLocalActivity, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(func(_ context.Context, _ *hooks.Hooks, _ *pipeline.Registry, _ logr.Logger, _ string, tinfo *TransferInfo) (*TransferInfo, error) {
			out := *tinfo
			out.PipelineConfig = &pipeline.Config{
				Name:               "pipeline",
				ID:                 "pipeline-id",
				TransferDir:        "/transfer-dir",
				TransferLocationID: "transfer-location-id",
				Recovery: pipeline.RecoveryConfig{
					ReconcileExistingAIP: true,
				},
			}
			return &out, nil
		}).Once()
	s.env.OnActivity(activities.AcquirePipelineActivityName, "pipeline").Return(nil).Once()
	s.env.OnActivity(setStatusInProgressLocalActivity, mock.Anything, mock.Anything, uint(12345), mock.Anything).Return(nil).Once()
	s.env.OnActivity(activities.BundleActivityName, &activities.BundleActivityParams{
		TransferDir:        "/transfer-dir",
		Key:                "key",
		TempFile:           "",
		StripTopLevelDir:   false,
		ExcludeHiddenFiles: false,
		IsDir:              false,
		BatchDir:           "/batch-dir",
		Unbag:              false,
	}).Return(&activities.BundleActivityResult{
		RelPath:  "key",
		FullPath: "/transfer-dir/key",
	}, nil).Once()
	s.env.OnActivity(activities.TransferActivityName, &activities.TransferActivityParams{
		PipelineName:       "pipeline",
		TransferLocationID: "transfer-location-id",
		RelPath:            "key",
		Name:               "key",
		ProcessingConfig:   "",
		TransferType:       "",
		Accession:          "",
	}).Return(&activities.TransferActivityResponse{
		TransferID: "transfer-id",
		PipelineID: "new-pipeline-id",
	}, nil).Once()
	s.env.OnActivity(updatePackageLocalActivity, mock.Anything, mock.Anything, mock.Anything, &updatePackageLocalActivityParams{
		CollectionID: uint(12345),
		Key:          "key",
		PipelineID:   "new-pipeline-id",
		TransferID:   "transfer-id",
		SIPID:        "",
		StoredAt:     time.Time{},
		Status:       collection.StatusInProgress,

		ProcessingConfig: new(""),
	}).Return(nil).Once()
	s.env.OnActivity(activities.PollTransferActivityName, &activities.PollTransferActivityParams{
		PipelineName: "pipeline",
		TransferID:   "transfer-id",
	}).Return("aip-id", nil).Once()
	s.env.OnActivity(updatePackageLocalActivity, mock.Anything, mock.Anything, mock.Anything, &updatePackageLocalActivityParams{
		CollectionID: uint(12345),
		Key:          "key",
		PipelineID:   "new-pipeline-id",
		TransferID:   "transfer-id",
		SIPID:        "aip-id",
		StoredAt:     time.Time{},
		Status:       collection.StatusInProgress,

		ProcessingConfig: new(""),
	}).Return(nil).Once()
	s.env.OnActivity(activities.PollIngestActivityName, &activities.PollIngestActivityParams{
		PipelineName: "pipeline",
		SIPID:        "aip-id",
	}).Return(time.Time{}, temporal.NewNonRetryableError(pollErr)).Once()
	s.env.OnActivity(activities.ReconcileStorageActivityName, &activities.ReconcileStorageActivityParams{
		PipelineName: "pipeline",
		AIPID:        "aip-id",
	}).Return(&activities.ReconcileStorageActivityResponse{
		Classification: reconciliation.ClassificationReplicatedComplete,
		Status:         reconciliation.StatusComplete,
		AIPStoredAt:    &aipStoredAt,
		CompletedAt:    &completedAt,
	}, nil).Once()
	s.env.OnActivity(updateReconciliationLocalActivity, mock.Anything, mock.Anything, mock.Anything, mock.MatchedBy(func(params *updateReconciliationLocalActivityParams) bool {
		return params.CollectionID == 12345 &&
			params.AIPStoredAt != nil &&
			params.CheckedAt != nil &&
			params.Status != nil &&
			*params.Status == string(reconciliation.StatusComplete) &&
			params.Error == nil &&
			params.AIPStoredAt.Format(time.RFC3339) == "2026-03-17T07:00:00Z"
	})).Return(nil).Once()
	s.env.OnActivity(releasePipelineLocalActivity, mock.Anything, mock.Anything, "pipeline").Return(nil).Once()
	s.env.OnActivity(activities.HidePackageActivityName, "transfer-id", "transfer", "pipeline", false).Return(nil).Once()
	s.env.OnActivity(activities.HidePackageActivityName, "aip-id", "ingest", "pipeline", false).Return(nil).Once()
	s.env.OnActivity(updatePackageLocalActivity, mock.Anything, mock.Anything, mock.Anything, &updatePackageLocalActivityParams{
		CollectionID: uint(12345),
		Key:          "key",
		PipelineID:   "new-pipeline-id",
		TransferID:   "transfer-id",
		SIPID:        "aip-id",
		StoredAt:     finalStoredAt,
		Status:       collection.StatusDone,

		ProcessingConfig: new(""),
	}).Return(nil).Once()

	s.env.ExecuteWorkflow(s.workflow.Execute, &collection.ProcessingWorkflowRequest{
		CollectionID: 12345,
		PipelineName: "pipeline",
		Key:          "key",
		BatchDir:     "/batch-dir",
	})

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
}

func (s *ProcessingWorkflowTestSuite) TestRecoveryEnabledFailsFastOnNotFoundAfterIngestFailure() {
	s.hooks.Hooks["hari"]["disabled"] = true
	s.hooks.Hooks["prod"]["disabled"] = true

	ctrl := gomock.NewController(s.T())
	pipelineRegistry, err := pipeline.NewPipelineRegistry(logr.Discard(), []pipeline.Config{
		{Name: "pipeline", ID: "pipeline-id"},
	}, nil, nil)
	s.Require().NoError(err)
	s.workflow = NewProcessingWorkflow(s.hooks, collectionfake.NewMockService(ctrl), pipelineRegistry, logr.Discard(), Config{})
	registerWorkflowActivityStubs(s.env)

	pollErr := errors.New("ingest poll failed")
	s.env.OnActivity(updatePackageLocalActivity, mock.Anything, mock.Anything, mock.Anything, &updatePackageLocalActivityParams{
		CollectionID: uint(12345),
		Key:          "key",
		PipelineID:   "",
		TransferID:   "",
		SIPID:        "",
		StoredAt:     time.Time{},
		Status:       collection.StatusQueued,
	}).Return(nil).Once()
	s.env.OnActivity(nha_activities.ParseNameLocalActivity, mock.Anything, "key").Return(&nha.NameInfo{}, nil).Once()
	s.env.OnActivity(loadConfigLocalActivity, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(func(_ context.Context, _ *hooks.Hooks, _ *pipeline.Registry, _ logr.Logger, _ string, tinfo *TransferInfo) (*TransferInfo, error) {
			out := *tinfo
			out.PipelineConfig = &pipeline.Config{
				Name:               "pipeline",
				ID:                 "pipeline-id",
				TransferDir:        "/transfer-dir",
				TransferLocationID: "transfer-location-id",
				Recovery: pipeline.RecoveryConfig{
					ReconcileExistingAIP: true,
				},
			}
			return &out, nil
		}).Once()
	s.env.OnActivity(activities.AcquirePipelineActivityName, "pipeline").Return(nil).Once()
	s.env.OnActivity(setStatusInProgressLocalActivity, mock.Anything, mock.Anything, uint(12345), mock.Anything).Return(nil).Once()
	s.env.OnActivity(activities.BundleActivityName, &activities.BundleActivityParams{
		TransferDir:        "/transfer-dir",
		Key:                "key",
		TempFile:           "",
		StripTopLevelDir:   false,
		ExcludeHiddenFiles: false,
		IsDir:              false,
		BatchDir:           "/batch-dir",
		Unbag:              false,
	}).Return(&activities.BundleActivityResult{
		RelPath:  "key",
		FullPath: "/transfer-dir/key",
	}, nil).Once()
	s.env.OnActivity(activities.TransferActivityName, &activities.TransferActivityParams{
		PipelineName:       "pipeline",
		TransferLocationID: "transfer-location-id",
		RelPath:            "key",
		Name:               "key",
		ProcessingConfig:   "",
		TransferType:       "",
		Accession:          "",
	}).Return(&activities.TransferActivityResponse{
		TransferID: "transfer-id",
		PipelineID: "new-pipeline-id",
	}, nil).Once()
	s.env.OnActivity(updatePackageLocalActivity, mock.Anything, mock.Anything, mock.Anything, &updatePackageLocalActivityParams{
		CollectionID: uint(12345),
		Key:          "key",
		PipelineID:   "new-pipeline-id",
		TransferID:   "transfer-id",
		SIPID:        "",
		StoredAt:     time.Time{},
		Status:       collection.StatusInProgress,

		ProcessingConfig: new(""),
	}).Return(nil).Once()
	s.env.OnActivity(activities.PollTransferActivityName, &activities.PollTransferActivityParams{
		PipelineName: "pipeline",
		TransferID:   "transfer-id",
	}).Return("aip-id", nil).Once()
	s.env.OnActivity(updatePackageLocalActivity, mock.Anything, mock.Anything, mock.Anything, &updatePackageLocalActivityParams{
		CollectionID: uint(12345),
		Key:          "key",
		PipelineID:   "new-pipeline-id",
		TransferID:   "transfer-id",
		SIPID:        "aip-id",
		StoredAt:     time.Time{},
		Status:       collection.StatusInProgress,

		ProcessingConfig: new(""),
	}).Return(nil).Once()
	s.env.OnActivity(activities.PollIngestActivityName, &activities.PollIngestActivityParams{
		PipelineName: "pipeline",
		SIPID:        "aip-id",
	}).Return(time.Time{}, temporal.NewNonRetryableError(pollErr)).Once()
	s.env.OnActivity(activities.ReconcileStorageActivityName, &activities.ReconcileStorageActivityParams{
		PipelineName: "pipeline",
		AIPID:        "aip-id",
	}).Return(&activities.ReconcileStorageActivityResponse{
		Classification: reconciliation.ClassificationNotFound,
		Status:         reconciliation.StatusPending,
	}, nil).Once()
	s.env.OnActivity(updateReconciliationLocalActivity, mock.Anything, mock.Anything, mock.Anything, mock.MatchedBy(func(params *updateReconciliationLocalActivityParams) bool {
		return params.CollectionID == 12345 &&
			params.AIPStoredAt == nil &&
			params.CheckedAt != nil &&
			params.Status != nil &&
			*params.Status == string(reconciliation.StatusPending) &&
			params.Error != nil &&
			strings.Contains(*params.Error, string(reconciliation.ClassificationNotFound)) &&
			strings.Contains(*params.Error, "ingest poll failed")
	})).Return(nil).Once()
	s.env.OnActivity(releasePipelineLocalActivity, mock.Anything, mock.Anything, "pipeline").Return(nil).Once()
	s.env.OnActivity(updatePackageLocalActivity, mock.Anything, mock.Anything, mock.Anything, &updatePackageLocalActivityParams{
		CollectionID: uint(12345),
		Key:          "key",
		PipelineID:   "new-pipeline-id",
		TransferID:   "transfer-id",
		SIPID:        "aip-id",
		StoredAt:     time.Time{},
		Status:       collection.StatusError,

		ProcessingConfig: new(""),
	}).Return(nil).Once()

	s.env.ExecuteWorkflow(s.workflow.Execute, &collection.ProcessingWorkflowRequest{
		CollectionID: 12345,
		PipelineName: "pipeline",
		Key:          "key",
		BatchDir:     "/batch-dir",
	})

	s.True(s.env.IsWorkflowCompleted())
	s.ErrorContains(s.env.GetWorkflowError(), "ingest failed and storage reconciliation returned not_found")
	s.True(temporal.NonRetryableError(s.env.GetWorkflowError()))
}

func (s *ProcessingWorkflowTestSuite) TestRecoveryEnabledFailsFastOnReplicatedPartialAfterIngestFailure() {
	s.hooks.Hooks["hari"]["disabled"] = true
	s.hooks.Hooks["prod"]["disabled"] = true

	ctrl := gomock.NewController(s.T())
	pipelineRegistry, err := pipeline.NewPipelineRegistry(logr.Discard(), []pipeline.Config{
		{Name: "pipeline", ID: "pipeline-id"},
	}, nil, nil)
	s.Require().NoError(err)
	s.workflow = NewProcessingWorkflow(s.hooks, collectionfake.NewMockService(ctrl), pipelineRegistry, logr.Discard(), Config{})
	registerWorkflowActivityStubs(s.env)

	pollErr := errors.New("ingest poll failed")

	s.env.OnActivity(updatePackageLocalActivity, mock.Anything, mock.Anything, mock.Anything, &updatePackageLocalActivityParams{
		CollectionID: uint(12345),
		Key:          "key",
		PipelineID:   "",
		TransferID:   "",
		SIPID:        "",
		StoredAt:     time.Time{},
		Status:       collection.StatusQueued,
	}).Return(nil).Once()
	s.env.OnActivity(nha_activities.ParseNameLocalActivity, mock.Anything, "key").Return(&nha.NameInfo{}, nil).Once()
	s.env.OnActivity(loadConfigLocalActivity, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(func(_ context.Context, _ *hooks.Hooks, _ *pipeline.Registry, _ logr.Logger, _ string, tinfo *TransferInfo) (*TransferInfo, error) {
			out := *tinfo
			out.PipelineConfig = &pipeline.Config{
				Name:               "pipeline",
				ID:                 "pipeline-id",
				TransferDir:        "/transfer-dir",
				TransferLocationID: "transfer-location-id",
				Recovery: pipeline.RecoveryConfig{
					ReconcileExistingAIP: true,
				},
			}
			return &out, nil
		}).Once()
	s.env.OnActivity(activities.AcquirePipelineActivityName, "pipeline").Return(nil).Once()
	s.env.OnActivity(setStatusInProgressLocalActivity, mock.Anything, mock.Anything, uint(12345), mock.Anything).Return(nil).Once()
	s.env.OnActivity(activities.BundleActivityName, &activities.BundleActivityParams{
		TransferDir:        "/transfer-dir",
		Key:                "key",
		TempFile:           "",
		StripTopLevelDir:   false,
		ExcludeHiddenFiles: false,
		IsDir:              false,
		BatchDir:           "/batch-dir",
		Unbag:              false,
	}).Return(&activities.BundleActivityResult{
		RelPath:  "key",
		FullPath: "/transfer-dir/key",
	}, nil).Once()
	s.env.OnActivity(activities.TransferActivityName, &activities.TransferActivityParams{
		PipelineName:       "pipeline",
		TransferLocationID: "transfer-location-id",
		RelPath:            "key",
		Name:               "key",
		ProcessingConfig:   "",
		TransferType:       "",
		Accession:          "",
	}).Return(&activities.TransferActivityResponse{
		TransferID: "transfer-id",
		PipelineID: "new-pipeline-id",
	}, nil).Once()
	s.env.OnActivity(updatePackageLocalActivity, mock.Anything, mock.Anything, mock.Anything, &updatePackageLocalActivityParams{
		CollectionID: uint(12345),
		Key:          "key",
		PipelineID:   "new-pipeline-id",
		TransferID:   "transfer-id",
		SIPID:        "",
		StoredAt:     time.Time{},
		Status:       collection.StatusInProgress,

		ProcessingConfig: new(""),
	}).Return(nil).Once()
	s.env.OnActivity(activities.PollTransferActivityName, &activities.PollTransferActivityParams{
		PipelineName: "pipeline",
		TransferID:   "transfer-id",
	}).Return("aip-id", nil).Once()
	s.env.OnActivity(updatePackageLocalActivity, mock.Anything, mock.Anything, mock.Anything, &updatePackageLocalActivityParams{
		CollectionID: uint(12345),
		Key:          "key",
		PipelineID:   "new-pipeline-id",
		TransferID:   "transfer-id",
		SIPID:        "aip-id",
		StoredAt:     time.Time{},
		Status:       collection.StatusInProgress,

		ProcessingConfig: new(""),
	}).Return(nil).Once()
	s.env.OnActivity(activities.PollIngestActivityName, &activities.PollIngestActivityParams{
		PipelineName: "pipeline",
		SIPID:        "aip-id",
	}).Return(time.Time{}, pollErr).Once()
	s.env.OnActivity(activities.ReconcileStorageActivityName, &activities.ReconcileStorageActivityParams{
		PipelineName: "pipeline",
		AIPID:        "aip-id",
	}).Return(&activities.ReconcileStorageActivityResponse{
		Classification: reconciliation.ClassificationReplicatedPartial,
		Status:         reconciliation.StatusPartial,
	}, nil).Once()
	s.env.OnActivity(updateReconciliationLocalActivity, mock.Anything, mock.Anything, mock.Anything, mock.MatchedBy(func(params *updateReconciliationLocalActivityParams) bool {
		return params.CollectionID == 12345 &&
			params.CheckedAt != nil &&
			params.Status != nil &&
			*params.Status == string(reconciliation.StatusPartial) &&
			params.Error != nil
	})).Return(nil).Once()
	s.env.OnActivity(releasePipelineLocalActivity, mock.Anything, mock.Anything, "pipeline").Return(nil).Once()
	s.env.OnActivity(updatePackageLocalActivity, mock.Anything, mock.Anything, mock.Anything, &updatePackageLocalActivityParams{
		CollectionID: uint(12345),
		Key:          "key",
		PipelineID:   "new-pipeline-id",
		TransferID:   "transfer-id",
		SIPID:        "aip-id",
		StoredAt:     time.Time{},
		Status:       collection.StatusError,

		ProcessingConfig: new(""),
	}).Return(nil).Once()

	s.env.ExecuteWorkflow(s.workflow.Execute, &collection.ProcessingWorkflowRequest{
		CollectionID: 12345,
		PipelineName: "pipeline",
		Key:          "key",
		BatchDir:     "/batch-dir",
	})

	s.True(s.env.IsWorkflowCompleted())
	s.ErrorContains(s.env.GetWorkflowError(), "ingest failed and storage reconciliation returned replicated_partial")
}

func TestProcessingWorkflow(t *testing.T) {
	suite.Run(t, new(ProcessingWorkflowTestSuite))
}

func TestProcessingWorkflowRejectsIncompleteBag(t *testing.T) {
	validator, err := bagit.NewValidator(bagit.WithDeferredRuntime(), bagit.WithTempCacheDir())
	assert.NilError(t, err)
	t.Cleanup(func() { assert.NilError(t, validator.Close()) })

	for _, tc := range []struct {
		name         string
		transferType string
		inPlace      bool
		archive      bool
		unbag        bool
	}{
		{name: "copied batch", transferType: "standard"},
		{name: "in-place batch", transferType: "unzipped bag", inPlace: true},
		{name: "watched archive with unbagging", transferType: "unzipped bag", archive: true, unbag: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// The empty hidden file tests manifest membership independently of
			// checksum verification and Payload-Oxum.
			const payload = "hello\n"
			source := fs.NewDir(t, "source", fs.WithDir("transfer",
				fs.WithFile("bagit.txt", "BagIt-Version: 1.0\nTag-File-Character-Encoding: UTF-8\n"),
				fs.WithFile("manifest-sha256.txt", fmt.Sprintf("%x  data/hello world.txt\n%x  data/.DS_Store\n", sha256.Sum256([]byte(payload)), sha256.Sum256(nil))),
				fs.WithDir("data", fs.WithFile("hello world.txt", payload), fs.WithFile(".DS_Store", "")),
			))
			defer source.Remove()
			transferDir := t.TempDir()
			if tc.inPlace {
				transferDir = source.Path()
			}
			registry, err := pipeline.NewPipelineRegistry(logr.Discard(), []pipeline.Config{{
				Name:        "pipeline",
				ID:          "pipeline-id",
				Capacity:    1,
				TransferDir: transferDir,
				Unbag:       tc.unbag,
				TransferPublisher: publisher.Config{
					Type:                  "sftp",
					Host:                  "example.invalid",
					User:                  "test",
					Password:              "test",
					InsecureIgnoreHostKey: true,
				},
			}}, nil, nil)
			assert.NilError(t, err)
			p, err := registry.ByName("pipeline")
			assert.NilError(t, err)

			ctrl := gomock.NewController(t)
			colsvc := collectionfake.NewMockService(ctrl)
			colsvc.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, c *collection.Collection) error {
				assert.Check(t, c.Status == collection.StatusQueued)
				c.ID = 12345
				return nil
			})
			colsvc.EXPECT().SetStatusInProgress(gomock.Any(), uint(12345), gomock.Any()).DoAndReturn(func(context.Context, uint, time.Time) error {
				_, inUse := p.Capacity()
				assert.Check(t, inUse == 1)
				return nil
			})
			colsvc.EXPECT().UpdateWorkflowStatus(gomock.Any(), uint(12345), "transfer", gomock.Any(), gomock.Any(),
				"", "", "", nil, collection.StatusError, time.Time{},
			).Return(nil)

			req := &collection.ProcessingWorkflowRequest{
				PipelineName:       "pipeline",
				BatchDir:           source.Path(),
				Key:                "transfer",
				IsDir:              true,
				ExcludeHiddenFiles: true,
				TransferType:       tc.transferType,
			}
			h := hooks.NewHooks(map[string]map[string]any{"hari": {"disabled": true}, "prod": {"disabled": true}})
			w := NewProcessingWorkflow(h, colsvc, registry, logr.Discard(), Config{ActivityHeartbeatTimeout: time.Minute})
			ts := &temporalsdk_testsuite.WorkflowTestSuite{}
			env := ts.NewTestWorkflowEnvironment()
			// The first real BagIt activity extracts the embedded interpreter.
			env.SetTestTimeout(time.Minute)
			env.SetWorkerOptions(temporalsdk_worker.Options{EnableSessionWorker: true})
			env.RegisterActivityWithOptions(activities.NewAcquirePipelineActivity(registry).Execute,
				temporalsdk_activity.RegisterOptions{Name: activities.AcquirePipelineActivityName})
			bundle := activities.NewBundleActivity(nil, validator)
			validate := activities.NewValidateTransferActivity(validator)
			var bundleAttempts, validateAttempts atomic.Int32
			env.RegisterActivityWithOptions(func(ctx context.Context, params *activities.BundleActivityParams) (*activities.BundleActivityResult, error) {
				bundleAttempts.Add(1)
				assert.Check(t, params.CheckBagBeforeUnbag == tc.unbag)
				assert.Check(t, params.RequireBag == (tc.unbag && tc.transferType == "unzipped bag"))
				return bundle.Execute(ctx, params)
			}, temporalsdk_activity.RegisterOptions{Name: activities.BundleActivityName})
			env.RegisterActivityWithOptions(func(ctx context.Context, params *activities.ValidateTransferActivityParams) error {
				validateAttempts.Add(1)
				assert.Check(t, params.CheckBagCompleteness)
				assert.Check(t, params.RequireBag == (tc.transferType == "unzipped bag"))
				return validate.Execute(ctx, params)
			}, temporalsdk_activity.RegisterOptions{Name: activities.ValidateTransferActivityName})
			env.RegisterActivityWithOptions(func(*activities.PublishTransferActivityParams) (*activities.PublishTransferActivityResult, error) {
				t.Error("invalid bag reached publishing")
				return nil, temporal.NewNonRetryableError(errors.New("unexpected publishing"))
			}, temporalsdk_activity.RegisterOptions{Name: activities.PublishTransferActivityName})
			env.RegisterActivityWithOptions(func(*activities.TransferActivityParams) (*activities.TransferActivityResponse, error) {
				t.Error("invalid bag reached Archivematica submission")
				return nil, temporal.NewNonRetryableError(errors.New("unexpected submission"))
			}, temporalsdk_activity.RegisterOptions{Name: activities.TransferActivityName})
			env.RegisterActivityWithOptions(activities.NewCleanUpActivity().Execute,
				temporalsdk_activity.RegisterOptions{Name: activities.CleanUpActivityName})

			var downloadPath, extractedPath string
			if tc.archive {
				req.BatchDir = ""
				req.WatcherName = "watcher"
				req.IsDir = false
				file, err := os.CreateTemp(t.TempDir(), "download-*.zip")
				assert.NilError(t, err)
				downloadPath = file.Name()
				archive := zip.NewWriter(file)
				assert.NilError(t, archive.AddFS(os.DirFS(source.Join("transfer"))))
				assert.NilError(t, archive.Close())
				assert.NilError(t, file.Close())
				env.RegisterActivityWithOptions(func(string, string, string) (string, error) {
					return downloadPath, nil
				}, temporalsdk_activity.RegisterOptions{Name: activities.DownloadActivityName})
				env.RegisterActivityWithOptions(func(ctx context.Context, params *archiveextract.Params) (*archiveextract.Result, error) {
					result, err := archiveextract.New(archiveextract.Config{}).Execute(ctx, params)
					if err == nil {
						extractedPath = result.ExtractPath
					}
					return result, err
				}, temporalsdk_activity.RegisterOptions{Name: archiveextract.Name})
			}
			env.ExecuteWorkflow(w.Execute, req)
			assert.Assert(t, env.IsWorkflowCompleted())
			assert.ErrorContains(t, env.GetWorkflowError(), "data/.DS_Store")
			assert.Assert(t, temporal.NonRetryableError(env.GetWorkflowError()))
			assert.Equal(t, bundleAttempts.Load(), int32(1))
			if tc.unbag {
				assert.Equal(t, validateAttempts.Load(), int32(0))
			} else {
				assert.Equal(t, validateAttempts.Load(), int32(1))
			}
			_, inUse := p.Capacity()
			assert.Equal(t, inUse, int64(0))
			_, err = os.Stat(source.Join("transfer", "data", "hello world.txt"))
			assert.NilError(t, err)
			if !tc.inPlace {
				_, err = os.Stat(source.Join("transfer", "data", ".DS_Store"))
				assert.NilError(t, err)
				entries, err := os.ReadDir(transferDir)
				assert.NilError(t, err)
				assert.Equal(t, len(entries), 0)
			}
			if tc.archive {
				assert.Assert(t, extractedPath != "")
				for _, path := range []string{downloadPath, extractedPath} {
					_, err := os.Stat(path)
					assert.ErrorIs(t, err, os.ErrNotExist)
				}
			}
		})
	}
}

func TestSessionHandlerBagValidationGuard(t *testing.T) {
	for _, tc := range []struct {
		name      string
		version   temporalsdk_workflow.Version
		checksums bool
		submitted bool
		published bool
		unbag     bool
	}{
		{name: "old history", version: temporalsdk_workflow.DefaultVersion},
		{name: "old history with checksum check", version: temporalsdk_workflow.DefaultVersion, checksums: true},
		{name: "already submitted with checksum check", version: 1, checksums: true, submitted: true},
		{name: "already published", version: 1, published: true},
		{name: "unbagged with checksum check", version: 1, checksums: true, unbag: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			colsvc := collectionfake.NewMockService(gomock.NewController(t))
			w := NewProcessingWorkflow(nil, colsvc, nil, logr.Discard(), Config{ActivityHeartbeatTimeout: time.Minute})
			ts := &temporalsdk_testsuite.WorkflowTestSuite{}
			env := ts.NewTestWorkflowEnvironment()
			env.SetWorkerOptions(temporalsdk_worker.Options{EnableSessionWorker: true})
			registerWorkflowActivityStubs(env)
			env.RegisterActivityWithOptions(func(*activities.ValidateTransferActivityParams) error {
				return temporal.NewNonRetryableError(errors.New("unexpected transfer validation"))
			}, temporalsdk_activity.RegisterOptions{Name: activities.ValidateTransferActivityName})
			env.OnGetVersion("reject-incomplete-bags", temporalsdk_workflow.DefaultVersion, 1).Return(tc.version).Once()
			env.OnActivity(setStatusInProgressLocalActivity, mock.Anything, mock.Anything, uint(12345), mock.Anything).Return(nil).Once()
			env.OnActivity(releasePipelineLocalActivity, mock.Anything, mock.Anything, "pipeline").Return(nil).Once()

			path := t.TempDir()
			validationConfig := validation.Config{ChecksumsCheckEnabled: tc.checksums}
			if tc.checksums {
				env.OnActivity(activities.ValidateTransferActivityName, &activities.ValidateTransferActivityParams{
					Config: validationConfig,
					Path:   path,
				}).Return(nil).Once()
			}
			wantErr := "submission reached"
			if tc.submitted {
				wantErr = "polling reached"
				env.OnActivity(updatePackageLocalActivity, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
				env.OnActivity(activities.PollTransferActivityName, mock.Anything).
					Return("", temporal.NewNonRetryableError(errors.New(wantErr))).Once()
			} else {
				env.OnActivity(activities.TransferActivityName, mock.MatchedBy(func(params *activities.TransferActivityParams) bool {
					return params.TransferType == "unzipped bag"
				})).
					Return(nil, temporal.NewNonRetryableError(errors.New(wantErr))).Once()
			}
			env.ExecuteWorkflow(func(ctx temporalsdk_workflow.Context) error {
				activityOpts := temporalsdk_workflow.WithActivityOptions(ctx, temporalsdk_workflow.ActivityOptions{StartToCloseTimeout: time.Minute})
				sessCtx, err := temporalsdk_workflow.CreateSession(activityOpts, &temporalsdk_workflow.SessionOptions{
					CreationTimeout: time.Minute, ExecutionTimeout: time.Hour, HeartbeatTimeout: time.Minute,
				})
				if err != nil {
					return err
				}
				tinfo := &TransferInfo{
					CollectionID:   12345,
					PipelineName:   "pipeline",
					PipelineConfig: &pipeline.Config{Unbag: tc.unbag},
					TransferType:   "unzipped bag",
					Bundle:         activities.BundleActivityResult{FullPath: path},
				}
				if tc.submitted {
					tinfo.TransferID = "existing-transfer"
				}
				if tc.published {
					tinfo.PipelineConfig.TransferPublisher.Type = "sftp"
					tinfo.PublishedTransfer.RelPath = "published/key"
				}
				return w.SessionHandler(sessCtx, 1, tinfo, nha.NameInfo{}, validationConfig, NewTimer(), nil, &collection.ProcessingWorkflowRequest{})
			})
			assert.ErrorContains(t, env.GetWorkflowError(), wantErr)
			env.AssertExpectations(t)
		})
	}
}

func buildHooks(t *testing.T, ctrl *gomock.Controller) *hooks.Hooks {
	t.Helper()

	return hooks.NewHooks(
		map[string]map[string]any{
			"prod": {"disabled": "false"},
			"hari": {"disabled": "false"},
		},
	)
}

func registerWorkflowActivityStubs(env *temporalsdk_testsuite.TestWorkflowEnvironment) {
	env.RegisterActivityWithOptions(func(string) error { return nil }, temporalsdk_activity.RegisterOptions{Name: activities.AcquirePipelineActivityName})
	env.RegisterActivityWithOptions(func(*activities.ReconcileStorageActivityParams) (*activities.ReconcileStorageActivityResponse, error) {
		return nil, nil
	}, temporalsdk_activity.RegisterOptions{Name: activities.ReconcileStorageActivityName})
	env.RegisterActivityWithOptions(func(*activities.BundleActivityParams) (*activities.BundleActivityResult, error) {
		return nil, nil
	}, temporalsdk_activity.RegisterOptions{Name: activities.BundleActivityName})
	env.RegisterActivityWithOptions(func(*activities.PublishTransferActivityParams) (*activities.PublishTransferActivityResult, error) {
		return nil, nil
	}, temporalsdk_activity.RegisterOptions{Name: activities.PublishTransferActivityName})
	env.RegisterActivityWithOptions(func(*activities.CleanUpPublishedTransferActivityParams) error {
		return nil
	}, temporalsdk_activity.RegisterOptions{Name: activities.CleanUpPublishedActivityName})
	env.RegisterActivityWithOptions(func(*activities.TransferActivityParams) (*activities.TransferActivityResponse, error) {
		return nil, nil
	}, temporalsdk_activity.RegisterOptions{Name: activities.TransferActivityName})
	env.RegisterActivityWithOptions(func(*activities.PollTransferActivityParams) (string, error) {
		return "", nil
	}, temporalsdk_activity.RegisterOptions{Name: activities.PollTransferActivityName})
	env.RegisterActivityWithOptions(func(*activities.PollIngestActivityParams) (time.Time, error) {
		return time.Time{}, nil
	}, temporalsdk_activity.RegisterOptions{Name: activities.PollIngestActivityName})
	env.RegisterActivityWithOptions(func(*activities.CleanUpActivityParams) error { return nil }, temporalsdk_activity.RegisterOptions{Name: activities.CleanUpActivityName})
	env.RegisterActivityWithOptions(func(string, string, string, bool) error { return nil }, temporalsdk_activity.RegisterOptions{Name: activities.HidePackageActivityName})
	env.RegisterActivityWithOptions(func(*nha_activities.UpdateHARIActivityParams) error { return nil }, temporalsdk_activity.RegisterOptions{Name: nha_activities.UpdateHARIActivityName})
	env.RegisterActivityWithOptions(func(*nha_activities.UpdateProductionSystemActivityParams) error { return nil }, temporalsdk_activity.RegisterOptions{Name: nha_activities.UpdateProductionSystemActivityName})
}

func TestTransferInfoProcessingConfiguration(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		tinfo            TransferInfo
		processingConfig string
	}{
		{
			tinfo: TransferInfo{
				ProcessingConfig: "automated",
				PipelineConfig: &pipeline.Config{
					ProcessingConfig: "default",
				},
			},
			processingConfig: "automated",
		},
		{
			tinfo: TransferInfo{
				ProcessingConfig: "automated",
				PipelineConfig:   nil,
			},
			processingConfig: "automated",
		},
		{
			tinfo: TransferInfo{
				ProcessingConfig: "",
				PipelineConfig:   nil,
			},
			processingConfig: "",
		},
		{
			tinfo: TransferInfo{
				ProcessingConfig: "",
				PipelineConfig: &pipeline.Config{
					ProcessingConfig: "default",
				},
			},
			processingConfig: "default",
		},
		{
			tinfo: TransferInfo{
				ProcessingConfig: "",
				PipelineConfig: &pipeline.Config{
					ProcessingConfig: "",
				},
			},
			processingConfig: "",
		},
	}
	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			t.Parallel()

			result := tc.tinfo.ProcessingConfiguration()
			assert.Equal(t, result, tc.processingConfig)
		})
	}
}

func TestReconciliationMessageIncludesIngestFailure(t *testing.T) {
	t.Parallel()

	msg := reconciliationMessage(reconciliation.ClassificationReplicatedPartial, errors.New("ingest poll failed"))
	if msg == nil {
		assert.Assert(t, false, "expected reconciliationMessage to return a message")
		return
	}

	assert.Assert(t, strings.Contains(*msg, "ingest failed and storage reconciliation returned replicated_partial"))
	assert.Assert(t, strings.Contains(*msg, "ingest poll failed"))
}

func TestShouldRetryPostIngestReconciliation(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		classification reconciliation.Classification
		want           bool
	}{
		{
			name:           "not found",
			classification: reconciliation.ClassificationNotFound,
			want:           true,
		},
		{
			name:           "indeterminate",
			classification: reconciliation.ClassificationIndeterminate,
			want:           true,
		},
		{
			name:           "replicated partial",
			classification: reconciliation.ClassificationReplicatedPartial,
			want:           false,
		},
		{
			name:           "local complete",
			classification: reconciliation.ClassificationLocalComplete,
			want:           false,
		},
		{
			name:           "replicated complete",
			classification: reconciliation.ClassificationReplicatedComplete,
			want:           false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := shouldRetryPostIngestReconciliation(tc.classification, nil)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestShouldNotRetryPostIngestReconciliationAfterIngestError(t *testing.T) {
	t.Parallel()

	got := shouldRetryPostIngestReconciliation(reconciliation.ClassificationNotFound, errors.New("ingest failed"))

	assert.Assert(t, !got)
}

func TestSetStoredAtFromReconciliationRequiresCompletedAt(t *testing.T) {
	t.Parallel()

	tinfo := &TransferInfo{}
	response := &activities.ReconcileStorageActivityResponse{
		Classification: reconciliation.ClassificationLocalComplete,
		Status:         reconciliation.StatusComplete,
	}

	err := setStoredAtFromReconciliation(tinfo, response)

	assert.ErrorContains(t, err, "storage reconciliation completed without a completion timestamp")
	assert.Assert(t, tinfo.StoredAt.IsZero())
}
