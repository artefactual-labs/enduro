package collection

import (
	"context"
	"fmt"
	"time"

	cce "github.com/artefactual-labs/enduro/internal/cadence"
	"github.com/artefactual-labs/enduro/internal/validation"
	"github.com/artefactual-labs/enduro/internal/watcher"

	"github.com/google/uuid"
	"go.uber.org/cadence/client"
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

	// Captured by the watcher, the event contains information about the
	// incoming dataset.
	Event *watcher.BlobEvent

	PipelineName string

	ValidationConfig validation.Config
}

func InitProcessingWorkflow(ctx context.Context, c client.Client, event *watcher.BlobEvent, pipelineName string, validationConfig validation.Config) error {
	req := &ProcessingWorkflowRequest{
		WorkflowID:       fmt.Sprintf("processing-workflow-%s", uuid.New().String()),
		Event:            event,
		PipelineName:     pipelineName,
		ValidationConfig: validationConfig,
	}

	return TriggerProcessingWorkflow(ctx, c, req)
}

func TriggerProcessingWorkflow(ctx context.Context, c client.Client, req *ProcessingWorkflowRequest) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	opts := client.StartWorkflowOptions{
		ID:                           req.WorkflowID,
		TaskList:                     cce.GlobalTaskListName,
		ExecutionStartToCloseTimeout: ProcessingWorkflowStartToCloseTimeout,
		WorkflowIDReusePolicy:        client.WorkflowIDReusePolicyAllowDuplicate,
	}
	_, err := c.StartWorkflow(ctx, opts, ProcessingWorkflowName, req)

	return err
}
