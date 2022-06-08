package batch

import (
	"context"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/mock"
	temporalapi_common "go.temporal.io/api/common/v1"
	temporalapi_enums "go.temporal.io/api/enums/v1"
	temporalapi_serviceerror "go.temporal.io/api/serviceerror"
	temporalapi_workflow "go.temporal.io/api/workflow/v1"
	temporalapi_workflowservice "go.temporal.io/api/workflowservice/v1"
	temporalsdk_mocks "go.temporal.io/sdk/mocks"
	"gotest.tools/v3/assert"

	goabatch "github.com/artefactual-labs/enduro/internal/api/gen/batch"
	"github.com/artefactual-labs/enduro/internal/collection"
)

var completedDirs = []string{"/tmp/xyz"}

func TestBatchServiceSubmit(t *testing.T) {
	ctx := context.Background()
	logger := logr.Discard()

	t.Run("Fails with empty or invalid parameters parameters", func(t *testing.T) {
		client := &temporalsdk_mocks.Client{}
		batchsvc := NewService(logger, client, completedDirs)

		_, err := batchsvc.Submit(ctx, &goabatch.SubmitPayload{})
		assert.Error(t, err, "error starting batch - path is empty")

		rp := "invalid-duration-format"
		_, err = batchsvc.Submit(ctx, &goabatch.SubmitPayload{Path: "/some/path", RetentionPeriod: &rp})
		assert.Error(t, err, "error starting batch - retention period format is invalid")
	})

	t.Run("Fails when workflow engine is unavailable", func(t *testing.T) {
		client := &temporalsdk_mocks.Client{}

		client.On(
			"ExecuteWorkflow",
			mock.AnythingOfType("*context.emptyCtx"),
			mock.AnythingOfType("internal.StartWorkflowOptions"),
			mock.AnythingOfType("string"),
			mock.AnythingOfType("batch.BatchWorkflowInput"),
		).Return(
			&temporalsdk_mocks.WorkflowRun{},
			&temporalapi_serviceerror.InvalidArgument{},
		)

		batchsvc := NewService(logger, client, completedDirs)
		_, err := batchsvc.Submit(ctx, &goabatch.SubmitPayload{Path: "asdf"})

		assert.ErrorContains(t, err, "error starting batch")
	})

	t.Run("Returns batch result", func(t *testing.T) {
		client := &temporalsdk_mocks.Client{}
		completedDir, retentionPeriod := "/tmp", "2h"

		workflowRun := &temporalsdk_mocks.WorkflowRun{}
		workflowRun.On("GetID").Return("batch-workflow")
		workflowRun.On("GetRunID").Return("some-run-id")

		dur := time.Duration(time.Hour * 2)
		client.On(
			"ExecuteWorkflow",
			mock.AnythingOfType("*context.emptyCtx"),
			mock.AnythingOfType("internal.StartWorkflowOptions"),
			"batch-workflow",
			BatchWorkflowInput{
				Path:            "/some/path",
				CompletedDir:    completedDir,
				RetentionPeriod: &dur,
			},
		).Return(
			workflowRun, nil,
		)

		batchsvc := NewService(logger, client, completedDirs)
		result, err := batchsvc.Submit(ctx, &goabatch.SubmitPayload{
			Path:            "/some/path",
			CompletedDir:    &completedDir,
			RetentionPeriod: &retentionPeriod,
		})

		assert.NilError(t, err)
		assert.DeepEqual(t, result, &goabatch.BatchResult{
			WorkflowID: "batch-workflow",
			RunID:      "some-run-id",
		})
	})
}

func TestBatchServiceStatus(t *testing.T) {
	ctx := context.Background()
	logger := logr.Discard()
	wid, rid := "batch-workflow", "some-run-id"

	t.Run("Fails if the workflow information is unavailable", func(t *testing.T) {
		client := &temporalsdk_mocks.Client{}
		client.On("DescribeWorkflowExecution", mock.AnythingOfType("*context.emptyCtx"), "batch-workflow", "").Return(nil, &temporalapi_serviceerror.Unavailable{})

		batchsvc := NewService(logger, client, completedDirs)
		_, err := batchsvc.Status(ctx)

		assert.ErrorIs(t, err, ErrBatchStatusUnavailable)
	})

	t.Run("Fails if the workflow information is incomplete", func(t *testing.T) {
		client := &temporalsdk_mocks.Client{}
		client.On("DescribeWorkflowExecution", mock.AnythingOfType("*context.emptyCtx"), "batch-workflow", "").Return(&temporalapi_workflowservice.DescribeWorkflowExecutionResponse{}, nil)

		batchsvc := NewService(logger, client, completedDirs)
		_, err := batchsvc.Status(ctx)

		assert.ErrorIs(t, err, ErrBatchStatusUnavailable)
	})

	t.Run("Identifies a non-running batch", func(t *testing.T) {
		client := &temporalsdk_mocks.Client{}
		client.On("DescribeWorkflowExecution", mock.AnythingOfType("*context.emptyCtx"), "batch-workflow", "").Return(nil, &temporalapi_serviceerror.NotFound{})

		batchsvc := NewService(logger, client, completedDirs)
		result, err := batchsvc.Status(ctx)

		assert.NilError(t, err)
		assert.DeepEqual(t, result, &goabatch.BatchStatusResult{
			Running: false,
		})
	})

	t.Run("Identifies a running batch", func(t *testing.T) {
		client := &temporalsdk_mocks.Client{}
		client.On("DescribeWorkflowExecution", mock.AnythingOfType("*context.emptyCtx"), "batch-workflow", "").Return(&temporalapi_workflowservice.DescribeWorkflowExecutionResponse{
			WorkflowExecutionInfo: &temporalapi_workflow.WorkflowExecutionInfo{
				Execution: &temporalapi_common.WorkflowExecution{
					WorkflowId: wid,
					RunId:      rid,
				},
				Status: temporalapi_enums.WORKFLOW_EXECUTION_STATUS_RUNNING,
			},
		}, nil)

		batchsvc := NewService(logger, client, completedDirs)
		result, err := batchsvc.Status(ctx)

		assert.NilError(t, err)
		assert.DeepEqual(t, result, &goabatch.BatchStatusResult{
			WorkflowID: &wid,
			RunID:      &rid,
			Running:    true,
		})
	})

	t.Run("Identifies a closed batch", func(t *testing.T) {
		client := &temporalsdk_mocks.Client{}
		client.On("DescribeWorkflowExecution", mock.AnythingOfType("*context.emptyCtx"), "batch-workflow", "").Return(&temporalapi_workflowservice.DescribeWorkflowExecutionResponse{
			WorkflowExecutionInfo: &temporalapi_workflow.WorkflowExecutionInfo{
				Execution: &temporalapi_common.WorkflowExecution{
					WorkflowId: wid,
					RunId:      rid,
				},
				Status: temporalapi_enums.WORKFLOW_EXECUTION_STATUS_COMPLETED,
			},
		}, nil)

		batchsvc := NewService(logger, client, completedDirs)
		result, err := batchsvc.Status(ctx)

		st := "completed"
		assert.NilError(t, err)
		assert.DeepEqual(t, result, &goabatch.BatchStatusResult{
			WorkflowID: &wid,
			RunID:      &rid,
			Running:    false,
			Status:     &st,
		})
	})
}

func TestBatchServiceHints(t *testing.T) {
	ctx := context.Background()
	logger := logr.Discard()
	client := &temporalsdk_mocks.Client{}

	batchsvc := NewService(logger, client, completedDirs)
	result, err := batchsvc.Hints(ctx)

	assert.NilError(t, err)
	assert.DeepEqual(t, result, &goabatch.BatchHintsResult{
		CompletedDirs: completedDirs,
	})
}

func TestBatchServiceInitProcessingWorkflow(t *testing.T) {
	ctx := context.Background()
	logger := logr.Discard()
	client := &temporalsdk_mocks.Client{}
	client.On(
		"ExecuteWorkflow",
		mock.AnythingOfType("*context.timerCtx"),
		mock.AnythingOfType("internal.StartWorkflowOptions"),
		mock.AnythingOfType("string"),
		mock.AnythingOfType("*collection.ProcessingWorkflowRequest"),
	).Return(
		nil,
		&temporalapi_serviceerror.Internal{},
	)

	batchsvc := NewService(logger, client, completedDirs)
	err := batchsvc.InitProcessingWorkflow(ctx, &collection.ProcessingWorkflowRequest{})

	assert.ErrorType(t, err, &temporalapi_serviceerror.Internal{})
}
