package batch

import (
	"context"
	"io/ioutil"
	"time"

	temporalsdk_temporal "go.temporal.io/sdk/temporal"
	temporalsdk_workflow "go.temporal.io/sdk/workflow"

	"github.com/artefactual-labs/enduro/internal/collection"
	"github.com/artefactual-labs/enduro/internal/temporal"
)

const (
	BatchWorkflowName = "batch-workflow"
	BatchWorkflowID   = "batch-workflow"
	BatchActivityName = "batch-activity"
)

type BatchProgress struct {
	CurrentID uint
}

type BatchWorkflowInput struct {
	Path            string
	CompletedDir    string
	RetentionPeriod *time.Duration
}

func BatchWorkflow(ctx temporalsdk_workflow.Context, params BatchWorkflowInput) error {
	opts := temporalsdk_workflow.WithActivityOptions(ctx, temporalsdk_workflow.ActivityOptions{
		StartToCloseTimeout: time.Hour * 24 * 365,
		WaitForCancellation: true,
		RetryPolicy: &temporalsdk_temporal.RetryPolicy{
			MaximumAttempts: 1,
		},
	})
	return temporalsdk_workflow.ExecuteActivity(opts, BatchActivityName, params).Get(opts, nil)
}

type BatchActivity struct {
	batchsvc Service
}

func NewBatchActivity(batchsvc Service) *BatchActivity {
	return &BatchActivity{
		batchsvc: batchsvc,
	}
}

func (a *BatchActivity) Execute(ctx context.Context, params BatchWorkflowInput) error {
	files, err := ioutil.ReadDir(params.Path)
	if err != nil {
		return temporal.NonRetryableError(err)
	}
	for _, file := range files {
		req := collection.ProcessingWorkflowRequest{
			BatchDir:        params.Path,
			Key:             file.Name(),
			IsDir:           file.IsDir(),
			CompletedDir:    params.CompletedDir,
			RetentionPeriod: params.RetentionPeriod,
		}
		_ = a.batchsvc.InitProcessingWorkflow(ctx, &req)
	}
	return nil
}
