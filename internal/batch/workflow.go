package batch

import (
	"context"
	"io/ioutil"
	"time"

	cadencesdk_workflow "go.uber.org/cadence/workflow"

	"github.com/artefactual-labs/enduro/internal/collection"
	wferrors "github.com/artefactual-labs/enduro/internal/workflow/errors"
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
	CompletedDir     string
	RetentionPeriod  *time.Duration
}

func BatchWorkflow(ctx cadencesdk_workflow.Context, params BatchWorkflowInput) error {
	opts := cadencesdk_workflow.WithActivityOptions(ctx, cadencesdk_workflow.ActivityOptions{
		ScheduleToStartTimeout: time.Hour * 24 * 365,
		StartToCloseTimeout:    time.Hour * 24 * 365,
		WaitForCancellation:    true,
	})
	return cadencesdk_workflow.ExecuteActivity(opts, BatchActivityName, params).Get(opts, nil)
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
	pipelines := []string{}
	if params.PipelineName != "" {
		pipelines = append(pipelines, params.PipelineName)
	}
	for _, file := range files {
		req := collection.ProcessingWorkflowRequest{
			BatchDir:         params.Path,
			Key:              file.Name(),
			IsDir:            file.IsDir(),
			PipelineNames:    pipelines,
			ProcessingConfig: params.ProcessingConfig,
			CompletedDir:     params.CompletedDir,
			RetentionPeriod:  params.RetentionPeriod,
		}
		_ = a.batchsvc.InitProcessingWorkflow(ctx, &req)
	}
	return nil
}
