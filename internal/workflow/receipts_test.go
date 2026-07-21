package workflow

import (
	"errors"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	temporalsdk_activity "go.temporal.io/sdk/activity"
	temporalsdk_testsuite "go.temporal.io/sdk/testsuite"
	temporalsdk_workflow "go.temporal.io/sdk/workflow"
	"go.uber.org/mock/gomock"
	"gotest.tools/v3/assert"

	"github.com/artefactual-labs/enduro/internal/collection"
	collectionfake "github.com/artefactual-labs/enduro/internal/collection/fake"
	"github.com/artefactual-labs/enduro/internal/nha"
	nha_activities "github.com/artefactual-labs/enduro/internal/nha/activities"
	"github.com/artefactual-labs/enduro/internal/pipeline"
)

func TestSendReceiptsStopsAfterAbandonDecision(t *testing.T) {
	env, w, params := newSendReceiptsTest(t)

	env.OnActivity(
		nha_activities.UpdateHARIActivityName,
		mock.Anything,
		hariParams(params),
	).Return(errors.New("failed")).Once()
	env.OnActivity(
		setStatusLocalActivity,
		mock.Anything,
		mock.Anything,
		params.CollectionID,
		collection.StatusPending,
	).Return(nil).Once()

	env.RegisterDelayedCallback(func() {
		env.UpdateWorkflowNoRejection(
			collection.ProcessingWorkflowDecisionUpdateName,
			"abandon-decision",
			t,
			collection.ProcessingWorkflowDecisionAbandon,
		)
	}, time.Second)

	executeSendReceiptsWorkflow(env, w, params)

	assert.Equal(t, env.IsWorkflowCompleted(), true)
	assert.ErrorContains(t, env.GetWorkflowError(), "error sending hari receipt: user abandoned")
	env.AssertExpectations(t)
}

func TestSendReceiptsRetriesAfterDecision(t *testing.T) {
	env, w, params := newSendReceiptsTest(t)
	w.hooks.Hooks["prod"]["disabled"] = true

	env.OnActivity(
		setStatusInProgressLocalActivity,
		mock.Anything,
		mock.Anything,
		params.CollectionID,
		time.Time{},
	).Return(nil).Once()
	env.OnActivity(
		nha_activities.UpdateHARIActivityName,
		mock.Anything,
		hariParams(params),
	).Return(errors.New("failed")).Once()
	env.OnActivity(
		nha_activities.UpdateHARIActivityName,
		mock.Anything,
		hariParams(params),
	).Return(nil).Once()
	env.OnActivity(
		setStatusLocalActivity,
		mock.Anything,
		mock.Anything,
		params.CollectionID,
		collection.StatusPending,
	).Return(nil).Once()

	env.RegisterDelayedCallback(func() {
		env.UpdateWorkflowNoRejection(
			collection.ProcessingWorkflowDecisionUpdateName,
			"retry-decision",
			t,
			collection.ProcessingWorkflowDecisionRetryOnce,
		)
	}, time.Second)

	executeSendReceiptsWorkflow(env, w, params)

	assert.Equal(t, env.IsWorkflowCompleted(), true)
	assert.NilError(t, env.GetWorkflowError())
	env.AssertExpectations(t)
}

func TestSendReceiptsRetriesOnlyFailedProductionReceipt(t *testing.T) {
	env, w, params := newSendReceiptsTest(t)

	env.OnActivity(
		nha_activities.UpdateHARIActivityName,
		mock.Anything,
		hariParams(params),
	).Return(nil).Once()
	env.OnActivity(
		nha_activities.UpdateProductionSystemActivityName,
		mock.Anything,
		prodParams(params),
	).Return(errors.New("failed")).Once()
	env.OnActivity(
		nha_activities.UpdateProductionSystemActivityName,
		mock.Anything,
		prodParams(params),
	).Return(nil).Once()
	env.OnActivity(
		setStatusLocalActivity,
		mock.Anything,
		mock.Anything,
		params.CollectionID,
		collection.StatusPending,
	).Return(nil).Once()
	env.OnActivity(
		setStatusInProgressLocalActivity,
		mock.Anything,
		mock.Anything,
		params.CollectionID,
		time.Time{},
	).Return(nil).Once()

	env.RegisterDelayedCallback(func() {
		env.UpdateWorkflowNoRejection(
			collection.ProcessingWorkflowDecisionUpdateName,
			"retry-prod-decision",
			t,
			collection.ProcessingWorkflowDecisionRetryOnce,
		)
	}, time.Second)

	executeSendReceiptsWorkflow(env, w, params)

	assert.Equal(t, env.IsWorkflowCompleted(), true)
	assert.NilError(t, env.GetWorkflowError())
	env.AssertExpectations(t)
}

func TestSendReceiptsAbandonsAfterDecisionTimeout(t *testing.T) {
	env, w, params := newSendReceiptsTest(t)
	w.hooks.Hooks["prod"]["disabled"] = true
	startedAt := env.Now()

	env.OnActivity(
		nha_activities.UpdateHARIActivityName,
		mock.Anything,
		hariParams(params),
	).Return(errors.New("failed")).Once()
	env.OnActivity(
		setStatusLocalActivity,
		mock.Anything,
		mock.Anything,
		params.CollectionID,
		collection.StatusPending,
	).Return(nil).Once()

	executeSendReceiptsWorkflow(env, w, params)

	assert.Equal(t, env.IsWorkflowCompleted(), true)
	assert.ErrorContains(t, env.GetWorkflowError(), "error sending hari receipt: user abandoned")
	assert.Equal(t, env.Now().Sub(startedAt) >= operatorDecisionTimeout, true)
	env.AssertExpectations(t)
}

func TestOperatorDecisionValidation(t *testing.T) {
	t.Run("Rejects decision when workflow is not awaiting one", func(t *testing.T) {
		env := new(temporalsdk_testsuite.WorkflowTestSuite).NewTestWorkflowEnvironment()
		var rejected error
		env.RegisterDelayedCallback(func() {
			env.UpdateWorkflow(
				collection.ProcessingWorkflowDecisionUpdateName,
				"early-decision",
				&temporalsdk_testsuite.TestUpdateCallback{
					OnReject: func(err error) { rejected = err },
					OnAccept: func() { t.Error("decision was accepted") },
				},
				collection.ProcessingWorkflowDecisionAbandon,
			)
		}, time.Second)

		env.ExecuteWorkflow(func(ctx temporalsdk_workflow.Context) error {
			if _, err := newOperatorDecisionHandler(ctx); err != nil {
				return err
			}
			return temporalsdk_workflow.Sleep(ctx, time.Minute)
		})

		assert.NilError(t, env.GetWorkflowError())
		assert.ErrorContains(t, rejected, "workflow is not awaiting an operator decision")
	})

	t.Run("Rejects unknown decision", func(t *testing.T) {
		env, w, params := newSendReceiptsTest(t)
		w.hooks.Hooks["prod"]["disabled"] = true
		var rejected error

		env.OnActivity(
			nha_activities.UpdateHARIActivityName,
			mock.Anything,
			hariParams(params),
		).Return(errors.New("failed")).Once()
		env.OnActivity(
			setStatusLocalActivity,
			mock.Anything,
			mock.Anything,
			params.CollectionID,
			collection.StatusPending,
		).Return(nil).Once()

		env.RegisterDelayedCallback(func() {
			env.UpdateWorkflow(
				collection.ProcessingWorkflowDecisionUpdateName,
				"unknown-decision",
				&temporalsdk_testsuite.TestUpdateCallback{
					OnReject: func(err error) { rejected = err },
					OnAccept: func() { t.Error("decision was accepted") },
				},
				collection.ProcessingWorkflowDecision("UNKNOWN"),
			)
		}, time.Second)
		env.RegisterDelayedCallback(func() {
			env.UpdateWorkflowNoRejection(
				collection.ProcessingWorkflowDecisionUpdateName,
				"abandon-decision",
				t,
				collection.ProcessingWorkflowDecisionAbandon,
			)
		}, 2*time.Second)

		executeSendReceiptsWorkflow(env, w, params)

		assert.ErrorContains(t, rejected, `unknown decision option "UNKNOWN"`)
		assert.ErrorContains(t, env.GetWorkflowError(), "user abandoned")
		env.AssertExpectations(t)
	})
}

func TestSendReceipts(t *testing.T) {
	env, w, params := newSendReceiptsTest(t)

	env.OnActivity(
		nha_activities.UpdateHARIActivityName,
		mock.Anything,
		hariParams(params),
	).Return(nil).Once()
	env.OnActivity(
		nha_activities.UpdateProductionSystemActivityName,
		mock.Anything,
		&nha_activities.UpdateProductionSystemActivityParams{
			StoredAt:     params.StoredAt,
			PipelineName: params.PipelineName,
			NameInfo:     params.NameInfo,
			FullPath:     params.FullPath,
		},
	).Return(nil).Once()

	executeSendReceiptsWorkflow(env, w, params)

	assert.Equal(t, env.IsWorkflowCompleted(), true)
	assert.NilError(t, env.GetWorkflowError())
	env.AssertExpectations(t)
}

func newSendReceiptsTest(t *testing.T) (*temporalsdk_testsuite.TestWorkflowEnvironment, *ProcessingWorkflow, *sendReceiptsParams) {
	t.Helper()

	env := new(temporalsdk_testsuite.WorkflowTestSuite).NewTestWorkflowEnvironment()
	h := buildHooks(t, gomock.NewController(t))
	colsvc := collectionfake.NewMockService(gomock.NewController(t))
	pipelineRegistry, _ := pipeline.NewPipelineRegistry(logr.Discard(), []pipeline.Config{}, nil, nil)

	nha_activities.UpdateHARIActivityName = uuid.New().String()
	env.RegisterActivityWithOptions(
		nha_activities.NewUpdateHARIActivity(h).Execute,
		temporalsdk_activity.RegisterOptions{Name: nha_activities.UpdateHARIActivityName},
	)

	nha_activities.UpdateProductionSystemActivityName = uuid.New().String()
	env.RegisterActivityWithOptions(
		nha_activities.NewUpdateProductionSystemActivity(h).Execute,
		temporalsdk_activity.RegisterOptions{Name: nha_activities.UpdateProductionSystemActivityName},
	)

	params := &sendReceiptsParams{
		SIPID:        "91e3ed2f-b798-4f4e-9133-74193f0d6a4f",
		StoredAt:     time.Now().UTC(),
		FullPath:     "/",
		PipelineName: "pipeline",
		NameInfo:     nha.NameInfo{},
		CollectionID: uint(12345),
	}

	return env, NewProcessingWorkflow(h, colsvc, pipelineRegistry, logr.Discard(), Config{}), params
}

func executeSendReceiptsWorkflow(
	env *temporalsdk_testsuite.TestWorkflowEnvironment,
	w *ProcessingWorkflow,
	params *sendReceiptsParams,
) {
	env.ExecuteWorkflow(func(ctx temporalsdk_workflow.Context, params *sendReceiptsParams) error {
		decisions, err := newOperatorDecisionHandler(ctx)
		if err != nil {
			return err
		}
		return w.sendReceipts(ctx, decisions, params)
	}, params)
}

func hariParams(params *sendReceiptsParams) *nha_activities.UpdateHARIActivityParams {
	return &nha_activities.UpdateHARIActivityParams{
		SIPID:        params.SIPID,
		StoredAt:     params.StoredAt,
		FullPath:     params.FullPath,
		PipelineName: params.PipelineName,
		NameInfo:     params.NameInfo,
	}
}

func prodParams(params *sendReceiptsParams) *nha_activities.UpdateProductionSystemActivityParams {
	return &nha_activities.UpdateProductionSystemActivityParams{
		StoredAt:     params.StoredAt,
		PipelineName: params.PipelineName,
		NameInfo:     params.NameInfo,
		FullPath:     params.FullPath,
	}
}
