// nolint:staticcheck
package workflow

import (
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	cadencesdk_activity "go.uber.org/cadence/activity"
	cadencesdk_testsuite "go.uber.org/cadence/testsuite"

	"github.com/artefactual-labs/enduro/internal/nha"
	nha_activities "github.com/artefactual-labs/enduro/internal/nha/activities"
)

// sendReceipts exits immediately after an activity error, ensuring that
// receipt delivery is halted once one delivery has failed.
func TestSendReceiptsSequentialBehavior(t *testing.T) {
	wts := cadencesdk_testsuite.WorkflowTestSuite{}
	env := wts.NewTestWorkflowEnvironment()
	m := buildManager(t, gomock.NewController(t))

	AsyncCompletionActivityName = uuid.New().String()
	cadencesdk_activity.RegisterWithOptions(NewAsyncCompletionActivity(m).Execute, cadencesdk_activity.RegisterOptions{Name: AsyncCompletionActivityName})

	nha_activities.UpdateHARIActivityName = uuid.New().String()
	cadencesdk_activity.RegisterWithOptions(nha_activities.NewUpdateHARIActivity(m).Execute, cadencesdk_activity.RegisterOptions{Name: nha_activities.UpdateHARIActivityName})

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

	env.ExecuteWorkflow(NewProcessingWorkflow(m).sendReceipts, &params)

	assert.True(t, env.IsWorkflowCompleted())
	assert.Equal(t, "error sending hari receipt: user abandoned", env.GetWorkflowError().Error())
	env.AssertExpectations(t)
}

func TestSendReceipts(t *testing.T) {
	wts := cadencesdk_testsuite.WorkflowTestSuite{}
	env := wts.NewTestWorkflowEnvironment()
	m := buildManager(t, gomock.NewController(t))

	nha_activities.UpdateHARIActivityName = uuid.New().String()
	cadencesdk_activity.RegisterWithOptions(nha_activities.NewUpdateHARIActivity(m).Execute, cadencesdk_activity.RegisterOptions{Name: nha_activities.UpdateHARIActivityName})

	nha_activities.UpdateProductionSystemActivityName = uuid.New().String()
	cadencesdk_activity.RegisterWithOptions(nha_activities.NewUpdateProductionSystemActivity(m).Execute, cadencesdk_activity.RegisterOptions{Name: nha_activities.UpdateProductionSystemActivityName})

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

	env.ExecuteWorkflow(NewProcessingWorkflow(m).sendReceipts, &params)

	assert.True(t, env.IsWorkflowCompleted())
	assertNilWorkflowError(t, env.GetWorkflowError())
	env.AssertExpectations(t)
}
