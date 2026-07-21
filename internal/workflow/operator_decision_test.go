package workflow

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	temporalsdk_activity "go.temporal.io/sdk/activity"
	temporalsdk_temporal "go.temporal.io/sdk/temporal"
	temporalsdk_testsuite "go.temporal.io/sdk/testsuite"
	temporalsdk_workflow "go.temporal.io/sdk/workflow"
	"go.uber.org/mock/gomock"
	"gotest.tools/v3/assert"

	"github.com/artefactual-labs/enduro/internal/collection"
	collectionfake "github.com/artefactual-labs/enduro/internal/collection/fake"
)

func TestRetryOnceOverridesActivityRetryPolicy(t *testing.T) {
	env := new(temporalsdk_testsuite.WorkflowTestSuite).NewTestWorkflowEnvironment()
	colsvc := collectionfake.NewMockService(gomock.NewController(t))
	activityName := uuid.NewString()
	var attempts atomic.Int32

	env.RegisterActivityWithOptions(func() error {
		attempts.Add(1)
		return errors.New("failed")
	}, temporalsdk_activity.RegisterOptions{Name: activityName})
	env.OnActivity(
		setStatusLocalActivity,
		mock.Anything,
		mock.Anything,
		uint(42),
		collection.StatusPending,
	).Return(nil).Twice()
	env.OnActivity(
		setStatusInProgressLocalActivity,
		mock.Anything,
		mock.Anything,
		uint(42),
		time.Time{},
	).Return(nil).Once()

	env.RegisterDelayedCallback(func() {
		env.UpdateWorkflowNoRejection(
			collection.ProcessingWorkflowDecisionUpdateName,
			"retry-once",
			t,
			collection.ProcessingWorkflowDecisionRetryOnce,
		)
	}, 10*time.Second)
	env.RegisterDelayedCallback(func() {
		env.UpdateWorkflowNoRejection(
			collection.ProcessingWorkflowDecisionUpdateName,
			"abandon",
			t,
			collection.ProcessingWorkflowDecisionAbandon,
		)
	}, 11*time.Second)

	env.ExecuteWorkflow(func(ctx temporalsdk_workflow.Context) error {
		decisions, err := newOperatorDecisionHandler(ctx)
		if err != nil {
			return err
		}
		return executeActivityWithOperatorDecision(
			ctx,
			decisions,
			colsvc,
			42,
			temporalsdk_workflow.ActivityOptions{
				StartToCloseTimeout: time.Minute,
				RetryPolicy: &temporalsdk_temporal.RetryPolicy{
					InitialInterval: time.Second,
					MaximumAttempts: 3,
				},
			},
			activityName,
		)
	})

	assert.ErrorContains(t, env.GetWorkflowError(), "user abandoned")
	assert.Equal(t, attempts.Load(), int32(4))
	env.AssertExpectations(t)
}

func TestOperatorDecisionPropagatesCancellation(t *testing.T) {
	env := new(temporalsdk_testsuite.WorkflowTestSuite).NewTestWorkflowEnvironment()
	colsvc := collectionfake.NewMockService(gomock.NewController(t))
	activityName := uuid.NewString()

	env.RegisterActivityWithOptions(
		func() error { return nil },
		temporalsdk_activity.RegisterOptions{Name: activityName},
	)

	env.ExecuteWorkflow(func(ctx temporalsdk_workflow.Context) error {
		decisions, err := newOperatorDecisionHandler(ctx)
		if err != nil {
			return err
		}

		activityCtx, cancel := temporalsdk_workflow.WithCancel(ctx)
		cancel()

		return executeActivityWithOperatorDecision(
			activityCtx,
			decisions,
			colsvc,
			42,
			temporalsdk_workflow.ActivityOptions{
				StartToCloseTimeout: time.Minute,
			},
			activityName,
		)
	})

	assert.ErrorContains(t, env.GetWorkflowError(), "canceled")
	env.AssertExpectations(t)
}

func TestOperatorDecisionWaitPropagatesCancellation(t *testing.T) {
	env := new(temporalsdk_testsuite.WorkflowTestSuite).NewTestWorkflowEnvironment()
	colsvc := collectionfake.NewMockService(gomock.NewController(t))
	activityName := uuid.NewString()

	env.RegisterActivityWithOptions(
		func() error { return errors.New("receipt failed") },
		temporalsdk_activity.RegisterOptions{Name: activityName},
	)
	env.OnActivity(
		setStatusLocalActivity,
		mock.Anything,
		mock.Anything,
		uint(42),
		collection.StatusPending,
	).Return(nil).Once()

	env.ExecuteWorkflow(func(ctx temporalsdk_workflow.Context) error {
		decisions, err := newOperatorDecisionHandler(ctx)
		if err != nil {
			return err
		}

		activityCtx, cancel := temporalsdk_workflow.WithCancel(ctx)
		temporalsdk_workflow.Go(ctx, func(ctx temporalsdk_workflow.Context) {
			_ = temporalsdk_workflow.Sleep(ctx, 10*time.Second)
			cancel()
		})

		return executeActivityWithOperatorDecision(
			activityCtx,
			decisions,
			colsvc,
			42,
			temporalsdk_workflow.ActivityOptions{
				StartToCloseTimeout: time.Minute,
				RetryPolicy: &temporalsdk_temporal.RetryPolicy{
					MaximumAttempts: 1,
				},
			},
			activityName,
		)
	})

	assert.ErrorContains(t, env.GetWorkflowError(), "canceled")
	env.AssertExpectations(t)
}

func TestRequiresOperatorDecision(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		err  error
		want bool
	}{
		"receipt failure": {
			err:  errors.New("receipt failed"),
			want: true,
		},
		"session failure": {
			err: temporalsdk_workflow.ErrSessionFailed,
		},
		"cancellation": {
			err: temporalsdk_temporal.NewCanceledError(),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, requiresOperatorDecision(tc.err), tc.want)
		})
	}
}
