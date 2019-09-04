package collection

import (
	"context"
	"fmt"
	"time"

	cce "github.com/artefactual-labs/enduro/internal/cadence"
	"github.com/artefactual-labs/enduro/internal/watcher"

	"github.com/google/uuid"
	"go.uber.org/cadence/client"
)

const (
	// Name of the collection processing workflow.
	ProcessingWorkflowName = "processing-workflow"

	// Maximum duration of the processing workflow. Cadence does not support
	// workflows with infinite duration for now, but high values are fine.
	// We consider a week more than enough.
	ProcessingWorkflowStartToCloseTimeout = time.Hour * 24 * 7
)

func InitProcessingWorkflow(ctx context.Context, c client.Client, event *watcher.BlobEvent) error {
	var workflowID = fmt.Sprintf("processing-workflow-%s", uuid.New().String())

	return TriggerProcessingWorkflow(ctx, c, event, 0, workflowID)
}

func TriggerProcessingWorkflow(ctx context.Context, c client.Client, event *watcher.BlobEvent, collectionID uint, workflowID string) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	opts := client.StartWorkflowOptions{
		ID:                           workflowID,
		TaskList:                     cce.GlobalTaskListName,
		ExecutionStartToCloseTimeout: ProcessingWorkflowStartToCloseTimeout,
		WorkflowIDReusePolicy:        client.WorkflowIDReusePolicyAllowDuplicate,
	}
	_, err := c.StartWorkflow(ctx, opts, ProcessingWorkflowName, event, collectionID)

	return err
}
