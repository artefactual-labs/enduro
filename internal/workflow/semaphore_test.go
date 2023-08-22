package workflow

import (
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	temporalsdk_activity "go.temporal.io/sdk/activity"
	temporalsdk_testsuite "go.temporal.io/sdk/testsuite"
	temporalsdk_workflow "go.temporal.io/sdk/workflow"
	"go.uber.org/mock/gomock"

	collectionfake "github.com/artefactual-labs/enduro/internal/collection/fake"
	"github.com/artefactual-labs/enduro/internal/pipeline"
	"github.com/artefactual-labs/enduro/internal/workflow/activities"
)

func TestSemaphoreAcquireRelease(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	wts := temporalsdk_testsuite.WorkflowTestSuite{}
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
	a := activities.NewAcquirePipelineActivity(registry)
	env.RegisterActivityWithOptions(
		a.Execute,
		temporalsdk_activity.RegisterOptions{
			Name: activities.AcquirePipelineActivityName,
		},
	)

	env.RegisterWorkflowWithOptions(
		func(ctx temporalsdk_workflow.Context) error {
			acquired, release, err := acquirePipeline(ctx, colsvc, registry, "am1", 12345)
			assert.Nil(t, err)
			assert.Equal(t, acquired, true)

			p, _ := registry.ByName("am1")
			size, cur := p.Capacity()
			assert.Equal(t, size, int64(1))
			assert.Equal(t, cur, int64(1))

			release(ctx)
			size, cur = p.Capacity()
			assert.Equal(t, size, int64(1))
			assert.Equal(t, cur, int64(0))

			return nil
		},
		temporalsdk_workflow.RegisterOptions{
			Name: "workflow",
		},
	)

	env.ExecuteWorkflow("workflow")

	assert.True(t, env.IsWorkflowCompleted())
	assert.Nil(t, env.GetWorkflowError())
}
