// nolint:staticcheck
package workflow

import (
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	temporalsdk_activity "go.temporal.io/sdk/activity"
	temporalsdk_testsuite "go.temporal.io/sdk/testsuite"
	"gotest.tools/v3/assert"

	"github.com/artefactual-labs/enduro/internal/nha"
	nha_activities "github.com/artefactual-labs/enduro/internal/nha/activities"
)

// sendReceipts exits immediately after an activity error, ensuring that
// receipt delivery is halted once one delivery has failed.
func TestSendReceiptsSequentialBehavior(t *testing.T) {
	wts := temporalsdk_testsuite.WorkflowTestSuite{}
	env := wts.NewTestWorkflowEnvironment()
	m := buildManager(t, gomock.NewController(t))

	AsyncCompletionActivityName = uuid.New().String() + "-async-completion"
	env.RegisterActivityWithOptions(NewAsyncCompletionActivity(m.Collection).Execute, temporalsdk_activity.RegisterOptions{Name: AsyncCompletionActivityName})

	nha_activities.UpdateHARIActivityName = uuid.New().String() + "-update-hary"
	env.RegisterActivityWithOptions(nha_activities.NewUpdateHARIActivity(m).Execute, temporalsdk_activity.RegisterOptions{Name: nha_activities.UpdateHARIActivityName})

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

	assert.Equal(t, env.IsWorkflowCompleted(), true)
	assert.ErrorContains(t, env.GetWorkflowError(), "error sending hari receipt: user abandoned")
	env.AssertExpectations(t)
}

func TestSendReceipts(t *testing.T) {
	wts := temporalsdk_testsuite.WorkflowTestSuite{}
	env := wts.NewTestWorkflowEnvironment()
	m := buildManager(t, gomock.NewController(t))

	nha_activities.UpdateHARIActivityName = uuid.New().String()
	env.RegisterActivityWithOptions(nha_activities.NewUpdateHARIActivity(m).Execute, temporalsdk_activity.RegisterOptions{Name: nha_activities.UpdateHARIActivityName})

	nha_activities.UpdateProductionSystemActivityName = uuid.New().String()
	env.RegisterActivityWithOptions(nha_activities.NewUpdateProductionSystemActivity(m).Execute, temporalsdk_activity.RegisterOptions{Name: nha_activities.UpdateProductionSystemActivityName})

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

	assert.Equal(t, env.IsWorkflowCompleted(), true)
	assert.NilError(t, env.GetWorkflowError())
	env.AssertExpectations(t)
}
