package workflow

import (
	"errors"
	"testing"
	"time"

	"github.com/artefactual-labs/enduro/internal/collection"
	collectionfake "github.com/artefactual-labs/enduro/internal/collection/fake"
	"github.com/artefactual-labs/enduro/internal/nha"
	nha_activities "github.com/artefactual-labs/enduro/internal/nha/activities"
	"github.com/artefactual-labs/enduro/internal/pipeline"
	"github.com/artefactual-labs/enduro/internal/watcher"
	watcherfake "github.com/artefactual-labs/enduro/internal/watcher/fake"
	"github.com/artefactual-labs/enduro/internal/workflow/manager"

	logrtesting "github.com/go-logr/logr/testing"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/cadence"
	cadenceactivity "go.uber.org/cadence/activity"
	cadencetestsuite "go.uber.org/cadence/testsuite"
	cadenceworkflow "go.uber.org/cadence/workflow"
	"go.uber.org/zap"
)

type ProcessingWorkflowTestSuite struct {
	suite.Suite
	cadencetestsuite.WorkflowTestSuite

	env *cadencetestsuite.TestWorkflowEnvironment

	manager *manager.Manager

	// Each test registers the workflow with a different name to avoid dups.
	workflow string
}

func (s *ProcessingWorkflowTestSuite) SetupTest() {
	s.SetLogger(zap.NewNop())
	s.env = s.NewTestWorkflowEnvironment()
	s.manager = buildManager(s.T(), gomock.NewController(s.T()))

	s.workflow = uuid.New().String()
	cadenceworkflow.RegisterWithOptions(NewProcessingWorkflow(s.manager).Execute, cadenceworkflow.RegisterOptions{Name: s.workflow})
}

func (s *ProcessingWorkflowTestSuite) AfterTest(suiteName, testName string) {
	s.env.AssertExpectations(s.T())
}

// Workflow ignores an error in parseName when NHA hooks are disabled.
func (s *ProcessingWorkflowTestSuite) TestParseErrorIsIgnored() {
	s.manager.Hooks["hari"]["disabled"] = true
	s.manager.Hooks["prod"]["disabled"] = true

	// Collection is persisted.
	s.env.OnActivity(createPackageLocalActivity, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(uint(12345), nil).Once()

	// parseName is executed with errors.
	s.env.OnActivity(nha_activities.ParseNameLocalActivity, mock.Anything, "key").Return(nil, errors.New("parse error")).Once()

	// loadConfig is executed (workflow continued), returning an error.
	s.env.OnActivity(loadConfigLocalActivity, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("pipeline is unavailable")).Once()

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
	s.env.ExecuteWorkflow(s.workflow, &collection.ProcessingWorkflowRequest{
		CollectionID: 0,
		Event: &watcher.BlobEvent{
			WatcherName:      "watcher",
			PipelineName:     "pipeline",
			RetentionPeriod:  &retentionPeriod,
			StripTopLevelDir: true,
			Key:              "key",
			Bucket:           "bucket",
		},
	})

	s.True(s.env.IsWorkflowCompleted())
	s.EqualError(s.env.GetWorkflowError(), "error loading configuration: pipeline is unavailable")
}

// Workflow does not ignore an error in parseName when NHA hooks are enabled.
func (s *ProcessingWorkflowTestSuite) TestParseError() {
	s.manager.Hooks["hari"]["disabled"] = false
	s.manager.Hooks["prod"]["disabled"] = false

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
	s.env.ExecuteWorkflow(s.workflow, &collection.ProcessingWorkflowRequest{
		CollectionID: 0,
		Event: &watcher.BlobEvent{
			WatcherName:      "watcher",
			PipelineName:     "pipeline",
			RetentionPeriod:  &retentionPeriod,
			StripTopLevelDir: true,
			Key:              "key",
			Bucket:           "bucket",
		},
	})

	s.True(s.env.IsWorkflowCompleted())
	s.EqualError(s.env.GetWorkflowError(), "error parsing transfer name: parse error")
}

func TestProcessingWorkflow(t *testing.T) {
	suite.Run(t, new(ProcessingWorkflowTestSuite))
}

// sendReceipts is a no-op when the status is "error".
func TestSendReceiptsNoop(t *testing.T) {
	wts := cadencetestsuite.WorkflowTestSuite{}
	wts.SetLogger(zap.NewNop())
	env := wts.NewTestWorkflowEnvironment()

	m := buildManager(t, gomock.NewController(t))

	wf := func(ctx cadenceworkflow.Context, hooks map[string]map[string]interface{}, params *sendReceiptsParams) error {
		return sendReceipts(ctx, hooks, params)
	}
	cadenceworkflow.Register(wf)

	env.ExecuteWorkflow(wf, m.Hooks, &sendReceiptsParams{
		Status: collection.StatusError,
	})

	assert.True(t, env.IsWorkflowCompleted())
	assertNilWorkflowError(t, env.GetWorkflowError())
	env.AssertExpectations(t)
}

// sendReceipts exits immediately after an activity error, ensuring that
// receipt delivery is halted once one delivery has failed.
func TestSendReceiptsSequentialBehavior(t *testing.T) {
	wts := cadencetestsuite.WorkflowTestSuite{}
	wts.SetLogger(zap.NewNop())
	env := wts.NewTestWorkflowEnvironment()

	m := buildManager(t, gomock.NewController(t))

	wf := func(ctx cadenceworkflow.Context, hooks map[string]map[string]interface{}, params *sendReceiptsParams) error {
		return sendReceipts(ctx, hooks, params)
	}
	cadenceworkflow.Register(wf)

	nha_activities.UpdateHARIActivityName = uuid.New().String()
	cadenceactivity.RegisterWithOptions(nha_activities.NewUpdateHARIActivity(m).Execute, cadenceactivity.RegisterOptions{Name: nha_activities.UpdateHARIActivityName})

	nha_activities.UpdateProductionSystemActivityName = uuid.New().String()
	cadenceactivity.RegisterWithOptions(nha_activities.NewUpdateProductionSystemActivity(m).Execute, cadenceactivity.RegisterOptions{Name: nha_activities.UpdateProductionSystemActivityName})

	params := sendReceiptsParams{
		SIPID:        "91e3ed2f-b798-4f4e-9133-74193f0d6a4f",
		StoredAt:     time.Now().UTC(),
		FullPath:     "/",
		PipelineName: "pipeline",
		NameInfo:     nha.NameInfo{},
		Status:       collection.StatusDone,
	}

	// Make HARI fail so the workflow returns immediately.
	env.OnActivity(
		nha_activities.UpdateHARIActivityName,
		mock.Anything,
		&nha_activities.UpdateHARIActivityParams{
			SIPID:        params.SIPID,
			StoredAt:     params.StoredAt,
			FullPath:     params.FullPath,
			PipelineName: params.PipelineName,
			NameInfo:     params.NameInfo,
		},
	).Return(errors.New("failed")).Once()

	env.ExecuteWorkflow(wf, m.Hooks, &params)

	assert.True(t, env.IsWorkflowCompleted())
	assert.Error(t, env.GetWorkflowError())
	env.AssertExpectations(t)
}

func TestSendReceipts(t *testing.T) {
	wts := cadencetestsuite.WorkflowTestSuite{}
	wts.SetLogger(zap.NewNop())
	env := wts.NewTestWorkflowEnvironment()

	m := buildManager(t, gomock.NewController(t))

	wf := func(ctx cadenceworkflow.Context, hooks map[string]map[string]interface{}, params *sendReceiptsParams) error {
		return sendReceipts(ctx, hooks, params)
	}
	cadenceworkflow.Register(wf)

	nha_activities.UpdateHARIActivityName = uuid.New().String()
	cadenceactivity.RegisterWithOptions(nha_activities.NewUpdateHARIActivity(m).Execute, cadenceactivity.RegisterOptions{Name: nha_activities.UpdateHARIActivityName})

	nha_activities.UpdateProductionSystemActivityName = uuid.New().String()
	cadenceactivity.RegisterWithOptions(nha_activities.NewUpdateProductionSystemActivity(m).Execute, cadenceactivity.RegisterOptions{Name: nha_activities.UpdateProductionSystemActivityName})

	params := sendReceiptsParams{
		SIPID:        "91e3ed2f-b798-4f4e-9133-74193f0d6a4f",
		StoredAt:     time.Now().UTC(),
		FullPath:     "/",
		PipelineName: "pipeline",
		NameInfo:     nha.NameInfo{},
		Status:       collection.StatusDone,
	}

	env.OnActivity(
		nha_activities.UpdateHARIActivityName,
		mock.Anything,
		&nha_activities.UpdateHARIActivityParams{
			SIPID:        params.SIPID,
			StoredAt:     params.StoredAt,
			FullPath:     params.FullPath,
			PipelineName: params.PipelineName,
			NameInfo:     params.NameInfo,
		},
	).Return(nil).Once()

	env.OnActivity(
		nha_activities.UpdateProductionSystemActivityName,
		mock.Anything,
		&nha_activities.UpdateProductionSystemActivityParams{
			StoredAt:     params.StoredAt,
			PipelineName: params.PipelineName,
			Status:       params.Status,
			NameInfo:     params.NameInfo,
		},
	).Return(nil).Once()

	env.ExecuteWorkflow(wf, m.Hooks, &params)

	assert.True(t, env.IsWorkflowCompleted())
	assertNilWorkflowError(t, env.GetWorkflowError())
	env.AssertExpectations(t)
}

func buildManager(t *testing.T, ctrl *gomock.Controller) *manager.Manager {
	t.Helper()

	return manager.NewManager(
		logrtesting.NullLogger{},
		collectionfake.NewMockService(ctrl),
		watcherfake.NewMockService(ctrl),
		&pipeline.Registry{},
		map[string]map[string]interface{}{
			"prod": {"disabled": "false"},
			"hari": {"disabled": "false"},
		},
	)
}
func assertNilWorkflowError(t *testing.T, err error) {
	t.Helper()

	if err == nil {
		return
	}

	if perr, ok := err.(*cadence.CustomError); ok {
		var details string
		perr.Details(&details)
		t.Fatal(details)
	} else {
		t.Fatal(err.Error())
	}

}
