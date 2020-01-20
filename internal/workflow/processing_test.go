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
	watcherfake "github.com/artefactual-labs/enduro/internal/watcher/fake"
	"github.com/artefactual-labs/enduro/internal/workflow/manager"

	logrtesting "github.com/go-logr/logr/testing"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/cadence"
	cadenceactivity "go.uber.org/cadence/activity"
	cadencetestsuite "go.uber.org/cadence/testsuite"
	cadenceworkflow "go.uber.org/cadence/workflow"
	"go.uber.org/zap"
)

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
