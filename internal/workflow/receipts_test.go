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
	"go.uber.org/mock/gomock"
	"gotest.tools/v3/assert"

	collectionfake "github.com/artefactual-labs/enduro/internal/collection/fake"
	"github.com/artefactual-labs/enduro/internal/nha"
	nha_activities "github.com/artefactual-labs/enduro/internal/nha/activities"
	"github.com/artefactual-labs/enduro/internal/pipeline"
)

// sendReceipts exits immediately after an activity error, ensuring that
// receipt delivery is halted once one delivery has failed.
func TestSendReceiptsSequentialBehavior(t *testing.T) {
	wts := temporalsdk_testsuite.WorkflowTestSuite{}
	env := wts.NewTestWorkflowEnvironment()
	h := buildHooks(t, gomock.NewController(t))
	ctrl := gomock.NewController(t)
	colsvc := collectionfake.NewMockService(ctrl)
	pipelineRegistry, _ := pipeline.NewPipelineRegistry(logr.Discard(), []pipeline.Config{})

	AsyncCompletionActivityName = uuid.New().String() + "-async-completion"
	env.RegisterActivityWithOptions(NewAsyncCompletionActivity(colsvc).Execute, temporalsdk_activity.RegisterOptions{Name: AsyncCompletionActivityName})

	nha_activities.UpdateHARIActivityName = uuid.New().String() + "-update-hary"
	env.RegisterActivityWithOptions(nha_activities.NewUpdateHARIActivity(h).Execute, temporalsdk_activity.RegisterOptions{Name: nha_activities.UpdateHARIActivityName})

	params := sendReceiptsParams{
		SIPID:        "91e3ed2f-b798-4f4e-9133-74193f0d6a4f",
		StoredAt:     time.Now().UTC(),
		FullPath:     "/",
		PipelineName: "pipeline",
		NameInfo:     nha.NameInfo{},
		CollectionID: uint(12345),
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

	env.OnActivity(
		AsyncCompletionActivityName,
		mock.Anything,
		uint(12345),
	).Return("ABANDON", nil).Once()

	env.ExecuteWorkflow(NewProcessingWorkflow(h, colsvc, pipelineRegistry, logr.Discard()).sendReceipts, &params)

	assert.Equal(t, env.IsWorkflowCompleted(), true)
	assert.ErrorContains(t, env.GetWorkflowError(), "error sending hari receipt: user abandoned")
	env.AssertExpectations(t)
}

func TestSendReceipts(t *testing.T) {
	wts := temporalsdk_testsuite.WorkflowTestSuite{}
	env := wts.NewTestWorkflowEnvironment()
	h := buildHooks(t, gomock.NewController(t))
	ctrl := gomock.NewController(t)
	colsvc := collectionfake.NewMockService(ctrl)
	pipelineRegistry, _ := pipeline.NewPipelineRegistry(logr.Discard(), []pipeline.Config{})

	nha_activities.UpdateHARIActivityName = uuid.New().String()
	env.RegisterActivityWithOptions(nha_activities.NewUpdateHARIActivity(h).Execute, temporalsdk_activity.RegisterOptions{Name: nha_activities.UpdateHARIActivityName})

	nha_activities.UpdateProductionSystemActivityName = uuid.New().String()
	env.RegisterActivityWithOptions(nha_activities.NewUpdateProductionSystemActivity(h).Execute, temporalsdk_activity.RegisterOptions{Name: nha_activities.UpdateProductionSystemActivityName})

	params := sendReceiptsParams{
		SIPID:        "91e3ed2f-b798-4f4e-9133-74193f0d6a4f",
		StoredAt:     time.Now().UTC(),
		FullPath:     "/",
		PipelineName: "pipeline",
		NameInfo:     nha.NameInfo{},
		CollectionID: uint(12345),
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
			NameInfo:     params.NameInfo,
			FullPath:     params.FullPath,
		},
	).Return(nil).Once()

	env.ExecuteWorkflow(NewProcessingWorkflow(h, colsvc, pipelineRegistry, logr.Discard()).sendReceipts, &params)

	assert.Equal(t, env.IsWorkflowCompleted(), true)
	assert.NilError(t, env.GetWorkflowError())
	env.AssertExpectations(t)
}
