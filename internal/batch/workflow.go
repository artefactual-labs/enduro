package batch

import (
	"context"
	"time"

	"go.uber.org/cadence/workflow"
)

const (
	BatchWorkflowName              = "batch-workflow"
	BatchWorkflowID                = "batch-workflow"
	BatchWorkflowStateQueryHandler = "batch-state"
	BatchActivityName              = "batch-activity"
)

type BatchProgress struct {
	CurrentID uint
}

type BatchWorkflowInput struct {
	Path         string
	PipelineName string
}

func BatchWorkflow(ctx workflow.Context, params BatchWorkflowInput) error {
	opts := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		ScheduleToStartTimeout: time.Hour * 24 * 365,
		StartToCloseTimeout:    time.Hour * 24 * 365,
		WaitForCancellation:    true,
		HeartbeatTimeout:       time.Second * 5,
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
	return nil
}
