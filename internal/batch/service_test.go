package batch

import (
	"context"
	"testing"

	goabatch "github.com/artefactual-labs/enduro/internal/api/gen/batch"

	logrtesting "github.com/go-logr/logr/testing"
	"github.com/stretchr/testify/mock"
	cadencemocks "go.uber.org/cadence/mocks"
	"go.uber.org/cadence/workflow"
	"gotest.tools/v3/assert"
)

func TestBatchServiceSubmitFailsWithEmptyParameters(t *testing.T) {
	ctx := context.Background()
	logger := logrtesting.NullLogger{}
	client := &cadencemocks.Client{}

	batchsvc := NewService(logger, client)
	_, err := batchsvc.Submit(ctx, &goabatch.SubmitPayload{Pipeline: "am"})
	assert.Error(t, err, "error starting batch - path is empty")

	_, err = batchsvc.Submit(ctx, &goabatch.SubmitPayload{Path: "/some/path"})
	assert.Error(t, err, "error starting batch - pipeline is empty")
}

func TestBatchServiceSubmit(t *testing.T) {
	ctx := context.Background()
	logger := logrtesting.NullLogger{}
	client := &cadencemocks.Client{}

	client.On("StartWorkflow", mock.Anything, mock.Anything, "batch-workflow", BatchWorkflowInput{Path: "/some/path", PipelineName: "am"}).Return(&workflow.Execution{ID: "batch-workflow", RunID: "some-run-id"}, nil)

	batchsvc := NewService(logger, client)
	result, err := batchsvc.Submit(ctx, &goabatch.SubmitPayload{
		Path:     "/some/path",
		Pipeline: "am",
	})
	assert.NilError(t, err)
	assert.Equal(t, result.WorkflowID, "batch-workflow")
	assert.Equal(t, result.RunID, "some-run-id")
}
