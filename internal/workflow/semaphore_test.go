package workflow

import (
	"testing"

	"github.com/go-logr/logr"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	cadencesdk_activity "go.uber.org/cadence/activity"
	cadencesdk_testsuite "go.uber.org/cadence/testsuite"
	cadencesdk_workflow "go.uber.org/cadence/workflow"

	collectionfake "github.com/artefactual-labs/enduro/internal/collection/fake"
	"github.com/artefactual-labs/enduro/internal/pipeline"
	"github.com/artefactual-labs/enduro/internal/workflow/activities"
	"github.com/artefactual-labs/enduro/internal/workflow/manager"
)

func TestSemaphoreAcquireRelease(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	wts := cadencesdk_testsuite.WorkflowTestSuite{}
	env := wts.NewTestWorkflowEnvironment()

	colsvc := collectionfake.NewMockService(ctrl)
	colsvc.
		EXPECT().
		SetStatusInProgress(gomock.Any(), gomock.Eq(uint(12345)), gomock.Any()).
		Return(nil)

	config := []pipeline.Config{
		{Name: "am1", Capacity: 1},
	}
	registry, _ := pipeline.NewPipelineRegistry(logr.Discard(), config)
	m := &manager.Manager{
		Pipelines:  registry,
		Collection: colsvc,
	}

	a := activities.NewAcquirePipelineActivity(m)
	env.RegisterActivityWithOptions(
		a.Execute,
		cadencesdk_activity.RegisterOptions{
			Name: activities.AcquirePipelineActivityName,
		},
	)

	env.RegisterWorkflowWithOptions(
		func(ctx cadencesdk_workflow.Context) error {
			acquired, release, err := acquirePipeline(ctx, m, "am1", 12345)
			assert.Nil(t, err)
			assert.Equal(t, acquired, true)

			p, _ := m.Pipelines.ByName("am1")
			size, cur := p.Capacity()
			assert.Equal(t, size, int64(1))
			assert.Equal(t, cur, int64(1))

			release(ctx, m, "am1")
			size, cur = p.Capacity()
			assert.Equal(t, size, int64(1))
			assert.Equal(t, cur, int64(0))

			return nil
		},
		cadencesdk_workflow.RegisterOptions{
			Name: "workflow",
		},
	)

	env.ExecuteWorkflow("workflow")

	assert.True(t, env.IsWorkflowCompleted())
	assert.Nil(t, env.GetWorkflowError())
}
