package workflow

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	temporalsdk_activity "go.temporal.io/sdk/activity"
	temporalsdk_testsuite "go.temporal.io/sdk/testsuite"
	temporalsdk_worker "go.temporal.io/sdk/worker"
	"go.uber.org/mock/gomock"
	"gotest.tools/v3/assert"

	"github.com/artefactual-labs/enduro/internal/collection"
	collectionfake "github.com/artefactual-labs/enduro/internal/collection/fake"
	"github.com/artefactual-labs/enduro/internal/nha"
	nha_activities "github.com/artefactual-labs/enduro/internal/nha/activities"
	"github.com/artefactual-labs/enduro/internal/pipeline"
	"github.com/artefactual-labs/enduro/internal/reconciliation"
	"github.com/artefactual-labs/enduro/internal/temporal"
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
	s.env.OnActivity(activities.TransferActivityName, &activities.TransferActivityParams{
		PipelineName:       "pipeline",
		TransferLocationID: "transfer-location-id",
		RelPath:            "key",
		Name:               "key",
		ProcessingConfig:   "",
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

func (s *ProcessingWorkflowTestSuite) TestRecoveryEnabledRetriesNotFoundAfterIngestFailure() {
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

			got := shouldRetryPostIngestReconciliation(tc.classification)
			assert.Equal(t, tc.want, got)
		})
	}
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
