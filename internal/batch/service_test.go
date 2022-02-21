package batch

import (
	"context"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/mock"
	cadencesdk_gen_shared "go.uber.org/cadence/.gen/go/shared"
	cadencesdk_mocks "go.uber.org/cadence/mocks"
	cadencesdk_workflow "go.uber.org/cadence/workflow"
	"gotest.tools/v3/assert"

	goabatch "github.com/artefactual-labs/enduro/internal/api/gen/batch"
	"github.com/artefactual-labs/enduro/internal/collection"
)

var completedDirs = []string{"/tmp/xyz"}

func TestBatchServiceSubmit(t *testing.T) {
	ctx := context.Background()
	logger := logr.Discard()
	pipeline := "am"

	t.Run("Fails with empty or invalid parameters parameters", func(t *testing.T) {
		client := &cadencesdk_mocks.Client{}
		batchsvc := NewService(logger, client, completedDirs)

		_, err := batchsvc.Submit(ctx, &goabatch.SubmitPayload{Pipeline: &pipeline})
		assert.Error(t, err, "error starting batch - path is empty")

		rp := "invalid-duration-format"
		_, err = batchsvc.Submit(ctx, &goabatch.SubmitPayload{Pipeline: &pipeline, Path: "/some/path", RetentionPeriod: &rp})
		assert.Error(t, err, "error starting batch - retention period format is invalid")
	})

	t.Run("Fails with empty parameters", func(t *testing.T) {
		client := &cadencesdk_mocks.Client{}
		processingConfig, completedDir, retentionPeriod := "default", "/tmp", "2h"

		dur := time.Duration(time.Hour * 2)
		client.On(
			"StartWorkflow", mock.Anything, mock.Anything, "batch-workflow",
			BatchWorkflowInput{
				Path:             "/some/path",
				PipelineName:     "am",
				ProcessingConfig: processingConfig,
				CompletedDir:     completedDir,
				RetentionPeriod:  &dur,
			},
		).Return(
			&cadencesdk_workflow.Execution{
				ID:    "batch-workflow",
				RunID: "some-run-id",
			}, nil,
		)

		batchsvc := NewService(logger, client, completedDirs)
		result, err := batchsvc.Submit(ctx, &goabatch.SubmitPayload{
			Path:             "/some/path",
			Pipeline:         &pipeline,
			ProcessingConfig: &processingConfig,
			CompletedDir:     &completedDir,
			RetentionPeriod:  &retentionPeriod,
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
		client := &cadencesdk_mocks.Client{}
		client.On("DescribeWorkflowExecution", mock.Anything, "batch-workflow", "").Return(nil, &cadencesdk_gen_shared.ServiceBusyError{})

		batchsvc := NewService(logger, client, completedDirs)
		_, err := batchsvc.Status(ctx)

		assert.ErrorIs(t, err, ErrBatchStatusUnavailable)
	})

	t.Run("Fails if the workflow information is incomplete", func(t *testing.T) {
		client := &cadencesdk_mocks.Client{}
		client.On("DescribeWorkflowExecution", mock.Anything, "batch-workflow", "").Return(&cadencesdk_gen_shared.DescribeWorkflowExecutionResponse{}, nil)

		batchsvc := NewService(logger, client, completedDirs)
		_, err := batchsvc.Status(ctx)

		assert.ErrorIs(t, err, ErrBatchStatusUnavailable)
	})

	t.Run("Identifies a non-running batch", func(t *testing.T) {
		client := &cadencesdk_mocks.Client{}
		client.On("DescribeWorkflowExecution", mock.Anything, "batch-workflow", "").Return(nil, &cadencesdk_gen_shared.EntityNotExistsError{})

		batchsvc := NewService(logger, client, completedDirs)
		result, err := batchsvc.Status(ctx)

		assert.NilError(t, err)
		assert.DeepEqual(t, result, &goabatch.BatchStatusResult{
			Running: false,
		})
	})

	t.Run("Identifies a running batch", func(t *testing.T) {
		client := &cadencesdk_mocks.Client{}
		client.On("DescribeWorkflowExecution", mock.Anything, "batch-workflow", "").Return(&cadencesdk_gen_shared.DescribeWorkflowExecutionResponse{
			WorkflowExecutionInfo: &cadencesdk_gen_shared.WorkflowExecutionInfo{
				Execution: &cadencesdk_gen_shared.WorkflowExecution{
					WorkflowId: &wid,
					RunId:      &rid,
				},
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
		client := &cadencesdk_mocks.Client{}
		client.On("DescribeWorkflowExecution", mock.Anything, "batch-workflow", "").Return(&cadencesdk_gen_shared.DescribeWorkflowExecutionResponse{
			WorkflowExecutionInfo: &cadencesdk_gen_shared.WorkflowExecutionInfo{
				Execution: &cadencesdk_gen_shared.WorkflowExecution{
					WorkflowId: &wid,
					RunId:      &rid,
				},
				CloseStatus: cadencesdk_gen_shared.WorkflowExecutionCloseStatusCompleted.Ptr(),
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
	client := &cadencesdk_mocks.Client{}

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
	client := &cadencesdk_mocks.Client{}
	client.On(
		"StartWorkflow",
		mock.AnythingOfType("*context.timerCtx"),
		mock.AnythingOfType("internal.StartWorkflowOptions"),
		mock.AnythingOfType("string"),
		mock.AnythingOfType("*collection.ProcessingWorkflowRequest"),
	).Return(
		nil,
		&cadencesdk_gen_shared.InternalServiceError{},
	)

	batchsvc := NewService(logger, client, completedDirs)
	err := batchsvc.InitProcessingWorkflow(ctx, &collection.ProcessingWorkflowRequest{})

	assert.ErrorType(t, err, &cadencesdk_gen_shared.InternalServiceError{})
}
