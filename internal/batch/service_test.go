package batch

import (
	"context"
	"testing"
	"time"

	goabatch "github.com/artefactual-labs/enduro/internal/api/gen/batch"
	"github.com/artefactual-labs/enduro/internal/collection"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/mock"
	"go.uber.org/cadence/.gen/go/shared"
	cadencemocks "go.uber.org/cadence/mocks"
	"go.uber.org/cadence/workflow"
	"gotest.tools/v3/assert"
)

var completedDirs = []string{"/tmp/xyz"}

func TestBatchServiceSubmit(t *testing.T) {
	ctx := context.Background()
	logger := logr.Discard()

	t.Run("Fails with empty or invalid parameters parameters", func(t *testing.T) {
		client := &cadencemocks.Client{}
		batchsvc := NewService(logger, client, completedDirs)

		_, err := batchsvc.Submit(ctx, &goabatch.SubmitPayload{Pipeline: "am"})
		assert.Error(t, err, "error starting batch - path is empty")

		_, err = batchsvc.Submit(ctx, &goabatch.SubmitPayload{Path: "/some/path"})
		assert.Error(t, err, "error starting batch - pipeline is empty")

		rp := "invalid-duration-format"
		_, err = batchsvc.Submit(ctx, &goabatch.SubmitPayload{Pipeline: "am", Path: "/some/path", RetentionPeriod: &rp})
		assert.Error(t, err, "error starting batch - retention period format is invalid")
	})

	t.Run("Fails with empty parameters", func(t *testing.T) {
		client := &cadencemocks.Client{}
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
			&workflow.Execution{
				ID:    "batch-workflow",
				RunID: "some-run-id",
			}, nil,
		)

		batchsvc := NewService(logger, client, completedDirs)
		result, err := batchsvc.Submit(ctx, &goabatch.SubmitPayload{
			Path:             "/some/path",
			Pipeline:         "am",
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
		client := &cadencemocks.Client{}
		client.On("DescribeWorkflowExecution", mock.Anything, "batch-workflow", "").Return(nil, &shared.ServiceBusyError{})

		batchsvc := NewService(logger, client, completedDirs)
		_, err := batchsvc.Status(ctx)

		assert.ErrorIs(t, err, ErrBatchStatusUnavailable)
	})

	t.Run("Fails if the workflow information is incomplete", func(t *testing.T) {
		client := &cadencemocks.Client{}
		client.On("DescribeWorkflowExecution", mock.Anything, "batch-workflow", "").Return(&shared.DescribeWorkflowExecutionResponse{}, nil)

		batchsvc := NewService(logger, client, completedDirs)
		_, err := batchsvc.Status(ctx)

		assert.ErrorIs(t, err, ErrBatchStatusUnavailable)
	})

	t.Run("Identifies a non-running batch", func(t *testing.T) {
		client := &cadencemocks.Client{}
		client.On("DescribeWorkflowExecution", mock.Anything, "batch-workflow", "").Return(nil, &shared.EntityNotExistsError{})

		batchsvc := NewService(logger, client, completedDirs)
		result, err := batchsvc.Status(ctx)

		assert.NilError(t, err)
		assert.DeepEqual(t, result, &goabatch.BatchStatusResult{
			Running: false,
		})
	})

	t.Run("Identifies a running batch", func(t *testing.T) {
		client := &cadencemocks.Client{}
		client.On("DescribeWorkflowExecution", mock.Anything, "batch-workflow", "").Return(&shared.DescribeWorkflowExecutionResponse{
			WorkflowExecutionInfo: &shared.WorkflowExecutionInfo{
				Execution: &shared.WorkflowExecution{
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
		client := &cadencemocks.Client{}
		client.On("DescribeWorkflowExecution", mock.Anything, "batch-workflow", "").Return(&shared.DescribeWorkflowExecutionResponse{
			WorkflowExecutionInfo: &shared.WorkflowExecutionInfo{
				Execution: &shared.WorkflowExecution{
					WorkflowId: &wid,
					RunId:      &rid,
				},
				CloseStatus: shared.WorkflowExecutionCloseStatusCompleted.Ptr(),
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
	client := &cadencemocks.Client{}

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
	client := &cadencemocks.Client{}
	client.On(
		"StartWorkflow",
		mock.AnythingOfType("*context.timerCtx"),
		mock.AnythingOfType("internal.StartWorkflowOptions"),
		mock.AnythingOfType("string"),
		mock.AnythingOfType("*collection.ProcessingWorkflowRequest"),
	).Return(
		nil,
		&shared.InternalServiceError{},
	)

	batchsvc := NewService(logger, client, completedDirs)
	err := batchsvc.InitProcessingWorkflow(ctx, &collection.ProcessingWorkflowRequest{})

	assert.ErrorType(t, err, &shared.InternalServiceError{})
}
