package collection

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	cadencesdk_client "go.uber.org/cadence/client"

	"github.com/artefactual-labs/enduro/internal/cadence"
	"github.com/artefactual-labs/enduro/internal/validation"
)

const (
	// Name of the collection processing workflow.
	ProcessingWorkflowName = "processing-workflow"

	// Maximum duration of the processing workflow. Cadence does not support
	// workflows with infinite duration for now, but high values are fine.
	// Ten years is the timeout we also use in activities (policies.go).
	ProcessingWorkflowStartToCloseTimeout = time.Hour * 24 * 365 * 10
)

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
}

func InitProcessingWorkflow(ctx context.Context, c cadencesdk_client.Client, req *ProcessingWorkflowRequest) error {
	if req.WorkflowID == "" {
		req.WorkflowID = fmt.Sprintf("processing-workflow-%s", uuid.New().String())
	}

	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	opts := cadencesdk_client.StartWorkflowOptions{
		ID:                           req.WorkflowID,
		TaskList:                     cadence.GlobalTaskListName,
		ExecutionStartToCloseTimeout: ProcessingWorkflowStartToCloseTimeout,
		WorkflowIDReusePolicy:        cadencesdk_client.WorkflowIDReusePolicyAllowDuplicate,
	}
	_, err := c.StartWorkflow(ctx, opts, ProcessingWorkflowName, req)

	return err
}
