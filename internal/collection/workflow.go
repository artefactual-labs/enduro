package collection

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	temporalsdk_api_enums "go.temporal.io/api/enums/v1"
	temporalsdk_client "go.temporal.io/sdk/client"

	"github.com/artefactual-labs/enduro/internal/validation"
)

// Name of the collection processing workflow.
const ProcessingWorkflowName = "processing-workflow"

type ProcessingWorkflowRequest struct {
	WorkflowID string `json:"-"`

	// The zero value represents a new collection. It can be used to indicate
	// an existing collection in retries.
	CollectionID uint

	// Name of the watcher that received this blob.
	WatcherName string

	// Pipelines that are available for processing. The workflow will choose one
	// (randomly picked for now). If the slice is empty, it will be
	// automatically populated from the list of all configured pipelines.
	PipelineNames []string

	// Period of time to schedule the deletion of the original blob from the
	// watched data source. nil means no deletion.
	RetentionPeriod *time.Duration

	// Directory where the transfer is moved to once processing has completed
	// successfully.
	CompletedDir string

	// Whether the top-level directory is meant to be stripped.
	StripTopLevelDir bool

	// Key of the blob.
	Key string

	// Whether the blob is a directory (fs watcher)
	IsDir bool

	// Batch directory that contains the blob.
	BatchDir string

	// Configuration for the validating the transfer.
	ValidationConfig validation.Config

	// Processing configuration name.
	ProcessingConfig string

	// Whether we reject duplicates based on name (key).
	RejectDuplicates bool
}

func InitProcessingWorkflow(ctx context.Context, tr trace.Tracer, c temporalsdk_client.Client, taskQueue string, req *ProcessingWorkflowRequest) error {
	_, span := tr.Start(ctx, "InitProcessingWorkflow")
	defer span.End()

	if req.WorkflowID == "" {
		req.WorkflowID = fmt.Sprintf("processing-workflow-%s", uuid.New().String())
	}

	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	opts := temporalsdk_client.StartWorkflowOptions{
		ID:                    req.WorkflowID,
		TaskQueue:             taskQueue,
		WorkflowIDReusePolicy: temporalsdk_api_enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE,
	}
	_, err := c.ExecuteWorkflow(ctx, opts, ProcessingWorkflowName, req)
	if err != nil {
		span.SetStatus(codes.Error, "ExecuteWorkflow failed")
		span.RecordError(err)
	}

	return err
}
