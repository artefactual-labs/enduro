package batch

import (
	"context"
	"errors"
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

var (
	completedDirs = []string{"/tmp/xyz"}
	taskQueue     = "global"
)

func TestBatchServiceSubmit(t *testing.T) {
	ctx := context.Background()
	logger := logr.Discard()
	pipeline := "am"

	t.Run("Fails with empty or invalid parameters parameters", func(t *testing.T) {
		client := &temporalsdk_mocks.Client{}
		batchsvc := NewService(logger, client, taskQueue, completedDirs)

		_, err := batchsvc.Submit(ctx, &goabatch.SubmitPayload{Pipeline: &pipeline})
		assert.Error(t, err, "error starting batch - path is empty")

		rp := "invalid-duration-format"
		_, err = batchsvc.Submit(ctx, &goabatch.SubmitPayload{Pipeline: &pipeline, Path: "/some/path", RetentionPeriod: &rp})
		assert.Error(t, err, "error starting batch - retention period format is invalid")
	})

	t.Run("Fails when workflow engine is unavailable", func(t *testing.T) {
		client := &temporalsdk_mocks.Client{}

		client.On(
			"ExecuteWorkflow",
			mock.AnythingOfType("context.backgroundCtx"),
			mock.AnythingOfType("internal.StartWorkflowOptions"),
			mock.AnythingOfType("string"),
			mock.AnythingOfType("batch.BatchWorkflowInput"),
		).Return(
			&temporalsdk_mocks.WorkflowRun{},
			&temporalapi_serviceerror.InvalidArgument{},
		)

		batchsvc := NewService(logger, client, taskQueue, completedDirs)
		_, err := batchsvc.Submit(ctx, &goabatch.SubmitPayload{Path: "asdf"})

		assert.ErrorContains(t, err, "error starting batch")
	})

	t.Run("Returns batch result", func(t *testing.T) {
		client := &temporalsdk_mocks.Client{}
		processingConfig, completedDir, retentionPeriod, transferType := "default", "/tmp", "2h", "standard"

		workflowRun := &temporalsdk_mocks.WorkflowRun{}
		workflowRun.On("GetID").Return("batch-workflow")
		workflowRun.On("GetRunID").Return("some-run-id")

		dur := time.Duration(time.Hour * 2)
		client.On(
			"ExecuteWorkflow",
			mock.AnythingOfType("context.backgroundCtx"),
			mock.AnythingOfType("internal.StartWorkflowOptions"),
			"batch-workflow",
			BatchWorkflowInput{
				Path:             "/some/path",
				PipelineName:     "am",
				ProcessingConfig: processingConfig,
				CompletedDir:     completedDir,
				RetentionPeriod:  &dur,
				TransferType:     transferType,
			},
		).Return(
			workflowRun, nil,
		)

		batchsvc := NewService(logger, client, taskQueue, completedDirs)
		result, err := batchsvc.Submit(ctx, &goabatch.SubmitPayload{
			Path:             "/some/path",
			Pipeline:         &pipeline,
			ProcessingConfig: &processingConfig,
			CompletedDir:     &completedDir,
			RetentionPeriod:  &retentionPeriod,
			TransferType:     &transferType,
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
		client.On("DescribeWorkflowExecution", mock.AnythingOfType("context.backgroundCtx"), "batch-workflow", "").Return(nil, &temporalapi_serviceerror.Unavailable{})

		batchsvc := NewService(logger, client, taskQueue, completedDirs)
		_, err := batchsvc.Status(ctx)

		assert.ErrorIs(t, err, ErrBatchStatusUnavailable)
	})

	t.Run("Fails if the workflow information is incomplete", func(t *testing.T) {
		client := &temporalsdk_mocks.Client{}
		client.On("DescribeWorkflowExecution", mock.AnythingOfType("context.backgroundCtx"), "batch-workflow", "").Return(&temporalapi_workflowservice.DescribeWorkflowExecutionResponse{}, nil)

		batchsvc := NewService(logger, client, taskQueue, completedDirs)
		_, err := batchsvc.Status(ctx)

		assert.ErrorIs(t, err, ErrBatchStatusUnavailable)
	})

	t.Run("Identifies a non-running batch", func(t *testing.T) {
		client := &temporalsdk_mocks.Client{}
		client.On("DescribeWorkflowExecution", mock.AnythingOfType("context.backgroundCtx"), "batch-workflow", "").Return(nil, &temporalapi_serviceerror.NotFound{})

		batchsvc := NewService(logger, client, taskQueue, completedDirs)
		result, err := batchsvc.Status(ctx)

		assert.NilError(t, err)
		assert.DeepEqual(t, result, &goabatch.BatchStatusResult{
			Running: false,
		})
	})

	t.Run("Identifies a running batch", func(t *testing.T) {
		client := &temporalsdk_mocks.Client{}
		client.On("DescribeWorkflowExecution", mock.AnythingOfType("context.backgroundCtx"), "batch-workflow", "").Return(&temporalapi_workflowservice.DescribeWorkflowExecutionResponse{
			WorkflowExecutionInfo: &temporalapi_workflow.WorkflowExecutionInfo{
				Execution: &temporalapi_common.WorkflowExecution{
					WorkflowId: wid,
					RunId:      rid,
				},
				Status: temporalapi_enums.WORKFLOW_EXECUTION_STATUS_RUNNING,
			},
		}, nil)

		batchsvc := NewService(logger, client, taskQueue, completedDirs)
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
		client.On("DescribeWorkflowExecution", mock.AnythingOfType("context.backgroundCtx"), "batch-workflow", "").Return(&temporalapi_workflowservice.DescribeWorkflowExecutionResponse{
			WorkflowExecutionInfo: &temporalapi_workflow.WorkflowExecutionInfo{
				Execution: &temporalapi_common.WorkflowExecution{
					WorkflowId: wid,
					RunId:      rid,
				},
				Status: temporalapi_enums.WORKFLOW_EXECUTION_STATUS_COMPLETED,
			},
		}, nil)

		batchsvc := NewService(logger, client, taskQueue, completedDirs)
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

	batchsvc := NewService(logger, client, taskQueue, completedDirs)
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
		temporalapi_serviceerror.NewInternal("message"),
	)

	batchsvc := NewService(logger, client, taskQueue, completedDirs)
	err := batchsvc.InitProcessingWorkflow(ctx, &collection.ProcessingWorkflowRequest{})

	var internalError *temporalapi_serviceerror.Internal
	assert.Assert(t, errors.As(err, &internalError) == true)
}
