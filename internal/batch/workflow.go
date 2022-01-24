package batch

import (
	"context"
	"io/ioutil"
	"time"

	wferrors "github.com/artefactual-labs/enduro/internal/workflow/errors"

	"go.uber.org/cadence/workflow"
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
	Path             string
	PipelineName     string
	ProcessingConfig string
}

func BatchWorkflow(ctx workflow.Context, params BatchWorkflowInput) error {
	opts := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		ScheduleToStartTimeout: time.Hour * 24 * 365,
		StartToCloseTimeout:    time.Hour * 24 * 365,
		WaitForCancellation:    true,
	})
	return workflow.ExecuteActivity(opts, BatchActivityName, params).Get(opts, nil)
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
		return wferrors.NonRetryableError(err)
	}
	for _, file := range files {
		_ = a.batchsvc.InitProcessingWorkflow(ctx, params.Path, file.Name(), file.IsDir(), params.PipelineName, params.ProcessingConfig)
	}
	return nil
}
