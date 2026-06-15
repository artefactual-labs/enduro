package batch

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/mock/gomock"
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
		BatchDir:         batchPath,
		Key:              "transfer1",
		IsDir:            true,
		PipelineName:     "am",
		ProcessingConfig: "automated",
	})
	serviceMock.EXPECT().InitProcessingWorkflow(ctx, &collection.ProcessingWorkflowRequest{
		BatchDir:         batchPath,
		Key:              "transfer2",
		IsDir:            true,
		PipelineName:     "am",
		ProcessingConfig: "automated",
	})

	// Execute the activity.
	err := a.Execute(ctx, BatchWorkflowInput{
		Path:             batchPath,
		PipelineName:     "am",
		ProcessingConfig: "automated",
	})
	assert.NilError(t, err)
}

func TestBatchActivityStartsProcessingWorkflowsForFilesAndDirectories(t *testing.T) {
	opts := []fs.PathOp{
		fs.WithFile("transfer1.zip", "contents"),
		fs.WithDir("transfer2",
			fs.WithFile("transfer2.txt", ""),
		),
	}
	tmpDir := fs.NewDir(t, "batch", opts...)
	batchPath := tmpDir.Path()
	defer tmpDir.Remove()

	ctrl := gomock.NewController(t)
	ctx := context.Background()
	serviceMock := batchfake.NewMockService(ctrl)
	a := NewBatchActivity(serviceMock)

	serviceMock.EXPECT().InitProcessingWorkflow(ctx, &collection.ProcessingWorkflowRequest{
		BatchDir:         batchPath,
		Key:              "transfer1.zip",
		IsDir:            false,
		PipelineName:     "am",
		ProcessingConfig: "automated",
	})
	serviceMock.EXPECT().InitProcessingWorkflow(ctx, &collection.ProcessingWorkflowRequest{
		BatchDir:         batchPath,
		Key:              "transfer2",
		IsDir:            true,
		PipelineName:     "am",
		ProcessingConfig: "automated",
	})

	err := a.Execute(ctx, BatchWorkflowInput{
		Path:             batchPath,
		PipelineName:     "am",
		ProcessingConfig: "automated",
	})
	assert.NilError(t, err)
}

func TestBatchActivityStartsProcessingWorkflowsForFilesAtDepth(t *testing.T) {
	opts := []fs.PathOp{
		fs.WithDir("lot1",
			fs.WithFile("transfer1.zip", "contents"),
			fs.WithFile(".DS_Store", "metadata"),
		),
		fs.WithDir("lot2",
			fs.WithFile("transfer2.zip", "contents"),
		),
	}
	tmpDir := fs.NewDir(t, "batch", opts...)
	batchPath := tmpDir.Path()
	defer tmpDir.Remove()

	ctrl := gomock.NewController(t)
	ctx := context.Background()
	serviceMock := batchfake.NewMockService(ctrl)
	a := NewBatchActivity(serviceMock)

	serviceMock.EXPECT().InitProcessingWorkflow(ctx, &collection.ProcessingWorkflowRequest{
		BatchDir:         tmpDir.Join("lot1"),
		Key:              "transfer1.zip",
		IsDir:            false,
		PipelineName:     "am",
		ProcessingConfig: "automated",
	})
	serviceMock.EXPECT().InitProcessingWorkflow(ctx, &collection.ProcessingWorkflowRequest{
		BatchDir:         tmpDir.Join("lot2"),
		Key:              "transfer2.zip",
		IsDir:            false,
		PipelineName:     "am",
		ProcessingConfig: "automated",
	})

	err := a.Execute(ctx, BatchWorkflowInput{
		Path:             batchPath,
		PipelineName:     "am",
		ProcessingConfig: "automated",
		Depth:            1,
	})
	assert.NilError(t, err)
}

func TestBatchActivityDoesNotDescendIntoDirectoryTransferAtDepth(t *testing.T) {
	opts := []fs.PathOp{
		fs.WithDir("transfer1",
			fs.WithFile("DSCF2073.JPG", "contents"),
		),
	}
	tmpDir := fs.NewDir(t, "batch", opts...)
	batchPath := tmpDir.Path()
	defer tmpDir.Remove()

	ctrl := gomock.NewController(t)
	ctx := context.Background()
	serviceMock := batchfake.NewMockService(ctrl)
	a := NewBatchActivity(serviceMock)

	serviceMock.EXPECT().InitProcessingWorkflow(ctx, &collection.ProcessingWorkflowRequest{
		BatchDir:         batchPath,
		Key:              "transfer1",
		IsDir:            true,
		PipelineName:     "am",
		ProcessingConfig: "automated",
	})

	err := a.Execute(ctx, BatchWorkflowInput{
		Path:             batchPath,
		PipelineName:     "am",
		ProcessingConfig: "automated",
		Depth:            0,
	})
	assert.NilError(t, err)
}

func TestBatchActivityStartsProcessingWorkflowsForMixedTransfersAtDepth(t *testing.T) {
	opts := []fs.PathOp{
		fs.WithDir("lot",
			fs.WithFile("transfer1.zip", "contents"),
			fs.WithDir("transfer2",
				fs.WithFile("DSCF2073.JPG", "contents"),
			),
		),
	}
	tmpDir := fs.NewDir(t, "batch", opts...)
	batchPath := tmpDir.Path()
	defer tmpDir.Remove()

	ctrl := gomock.NewController(t)
	ctx := context.Background()
	serviceMock := batchfake.NewMockService(ctrl)
	a := NewBatchActivity(serviceMock)

	serviceMock.EXPECT().InitProcessingWorkflow(ctx, &collection.ProcessingWorkflowRequest{
		BatchDir:         tmpDir.Join("lot"),
		Key:              "transfer1.zip",
		IsDir:            false,
		PipelineName:     "am",
		ProcessingConfig: "automated",
	})
	serviceMock.EXPECT().InitProcessingWorkflow(ctx, &collection.ProcessingWorkflowRequest{
		BatchDir:         tmpDir.Join("lot"),
		Key:              "transfer2",
		IsDir:            true,
		PipelineName:     "am",
		ProcessingConfig: "automated",
	})

	err := a.Execute(ctx, BatchWorkflowInput{
		Path:             batchPath,
		PipelineName:     "am",
		ProcessingConfig: "automated",
		Depth:            1,
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
		Path:         "/non/existent/path",
		PipelineName: "am",
	})
	assert.ErrorContains(t, err, "no such file or directory")
}

func TestBatchActivityFailsWhenProcessingWorkflowInitFails(t *testing.T) {
	opts := []fs.PathOp{
		fs.WithDir("transfer1",
			fs.WithFile("transfer1.txt", ""),
		),
	}
	tmpDir := fs.NewDir(t, "batch", opts...)
	batchPath := tmpDir.Path()
	defer tmpDir.Remove()

	ctrl := gomock.NewController(t)
	ctx := context.Background()
	serviceMock := batchfake.NewMockService(ctrl)
	a := NewBatchActivity(serviceMock)

	serviceMock.EXPECT().InitProcessingWorkflow(ctx, &collection.ProcessingWorkflowRequest{
		BatchDir:         batchPath,
		Key:              "transfer1",
		IsDir:            true,
		PipelineName:     "am",
		ProcessingConfig: "automated",
	}).Return(errors.New("workflow start failed"))

	err := a.Execute(ctx, BatchWorkflowInput{
		Path:             batchPath,
		PipelineName:     "am",
		ProcessingConfig: "automated",
	})
	assert.ErrorContains(t, err, "workflow start failed")
}
