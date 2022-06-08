package batch

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"

	batchfake "github.com/artefactual-labs/enduro/internal/batch/fake"
	"github.com/artefactual-labs/enduro/internal/collection"
)

func TestBatchActivityStartsProcessingWorkflows(t *testing.T) {
	// Create a temporary batch directory with two subdirectories.
	opts := []fs.PathOp{
		fs.WithDir("transfer1",
			fs.WithFile("transfer1.txt", ""),
		),
		fs.WithDir("transfer2",
			fs.WithFile("transfer2.txt", ""),
		),
	}
	tmpDir := fs.NewDir(t, "batch", opts...)
	batchPath := tmpDir.Path()
	defer tmpDir.Remove()

	// Set up the activity
	ctrl := gomock.NewController(t)
	ctx := context.Background()
	serviceMock := batchfake.NewMockService(ctrl)
	a := NewBatchActivity(serviceMock)

	// Expectations: the activity starts a processing workflow for each subdirectory.
	serviceMock.EXPECT().InitProcessingWorkflow(ctx, &collection.ProcessingWorkflowRequest{
		BatchDir: batchPath,
		Key:      "transfer1",
		IsDir:    true,
	})
	serviceMock.EXPECT().InitProcessingWorkflow(ctx, &collection.ProcessingWorkflowRequest{
		BatchDir: batchPath,
		Key:      "transfer2",
		IsDir:    true,
	})

	// Execute the activity.
	err := a.Execute(ctx, BatchWorkflowInput{
		Path: batchPath,
	})
	assert.NilError(t, err)
}

func TestBatchActivityFailsWithBogusBatchPath(t *testing.T) {
	// Set up the activity
	ctrl := gomock.NewController(t)
	ctx := context.Background()
	serviceMock := batchfake.NewMockService(ctrl)
	a := NewBatchActivity(serviceMock)

	// Execute the activity passing a bogus path.
	err := a.Execute(ctx, BatchWorkflowInput{
		Path: "/non/existent/path",
	})
	assert.Error(t, err, "non retryable error: open /non/existent/path: no such file or directory")
}
