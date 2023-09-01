package batch

import (
	"context"
	"io/fs"
	"path/filepath"
	"strings"
	"time"

	temporalsdk_temporal "go.temporal.io/sdk/temporal"
	temporalsdk_workflow "go.temporal.io/sdk/workflow"

	"github.com/artefactual-labs/enduro/internal/collection"
	"github.com/artefactual-labs/enduro/internal/metadata"
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
	Path             string
	PipelineName     string
	ProcessingConfig string
	CompletedDir     string
	RetentionPeriod  *time.Duration
	RejectDuplicates bool
	TransferType     string
	MetadataConfig   metadata.Config
	Depth            int32
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
	pipelines := []string{}
	if params.PipelineName != "" {
		pipelines = append(pipelines, params.PipelineName)
	}

	if params.Depth < 0 {
		params.Depth = 0
	}

	root := params.Path
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}

		if rel == "." {
			return nil // Ignore root.
		}

		depth := len(strings.Split(rel, string(filepath.Separator))) - 1
		if depth != int(params.Depth) {
			return nil // Keep walking.
		}

		req := collection.ProcessingWorkflowRequest{
			BatchDir:         filepath.Dir(path),
			Key:              entry.Name(),
			IsDir:            entry.IsDir(),
			PipelineNames:    pipelines,
			ProcessingConfig: params.ProcessingConfig,
			CompletedDir:     params.CompletedDir,
			RetentionPeriod:  params.RetentionPeriod,
			RejectDuplicates: params.RejectDuplicates,
			TransferType:     params.TransferType,
			MetadataConfig:   params.MetadataConfig,
		}

		_ = a.batchsvc.InitProcessingWorkflow(ctx, &req)

		return fs.SkipDir
	})
	if err != nil {
		return temporal.NewNonRetryableError(err)
	}

	return nil
}
