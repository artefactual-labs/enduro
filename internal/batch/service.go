package batch

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	goabatch "github.com/artefactual-labs/enduro/internal/api/gen/batch"
	"github.com/artefactual-labs/enduro/internal/cadence"
	"github.com/artefactual-labs/enduro/internal/collection"
	"github.com/artefactual-labs/enduro/internal/validation"
	"github.com/go-logr/logr"
	"go.uber.org/cadence/.gen/go/shared"
	cadenceclient "go.uber.org/cadence/client"
)

var ErrBatchStatusUnavailable = errors.New("batch status unavailable")

//go:generate mockgen  -destination=./fake/mock_batch.go -package=fake github.com/artefactual-labs/enduro/internal/batch Service

type Service interface {
	Submit(context.Context, *goabatch.SubmitPayload) (res *goabatch.BatchResult, err error)
	Status(context.Context) (res *goabatch.BatchStatusResult, err error)
	InitProcessingWorkflow(ctx context.Context, batchDir, key, pipelineName string) error
}

type batchImpl struct {
	logger logr.Logger
	cc     cadenceclient.Client
}

var _ Service = (*batchImpl)(nil)

func NewService(logger logr.Logger, cc cadenceclient.Client) *batchImpl {
	return &batchImpl{
		logger: logger,
		cc:     cc,
	}
}

func (s *batchImpl) Submit(ctx context.Context, payload *goabatch.SubmitPayload) (*goabatch.BatchResult, error) {
	if payload.Path == "" {
		return nil, goabatch.MakeNotAvailable(
			fmt.Errorf("error starting batch - path is empty"),
		)
	}
	var pipelineName string
	if payload.Pipeline != nil {
		pipelineName = *payload.Pipeline
	}
	input := BatchWorkflowInput{
		Path:         payload.Path,
		PipelineName: pipelineName,
	}
	opts := cadenceclient.StartWorkflowOptions{
		ID:                              BatchWorkflowID,
		WorkflowIDReusePolicy:           cadenceclient.WorkflowIDReusePolicyAllowDuplicate,
		TaskList:                        cadence.GlobalTaskListName,
		DecisionTaskStartToCloseTimeout: time.Second * 10,
		ExecutionStartToCloseTimeout:    time.Hour,
	}
	exec, err := s.cc.StartWorkflow(ctx, opts, BatchWorkflowName, input)
	if err != nil {
		switch err := err.(type) {
		case *shared.WorkflowExecutionAlreadyStartedError:
			return nil, goabatch.MakeNotAvailable(
				fmt.Errorf("error starting batch - operation is already in progress (workflowID=%s runID=%s)",
					BatchWorkflowID, *err.RunId))
		default:
			s.logger.Info("error starting batch", "err", err)
			return nil, fmt.Errorf("error starting batch")
		}
	}
	result := &goabatch.BatchResult{
		WorkflowID: exec.ID,
		RunID:      exec.RunID,
	}
	return result, nil
}

func (s *batchImpl) Status(ctx context.Context) (*goabatch.BatchStatusResult, error) {
	result := &goabatch.BatchStatusResult{}
	resp, err := s.cc.DescribeWorkflowExecution(ctx, BatchWorkflowID, "")
	if err != nil {
		switch err := err.(type) {
		case *shared.EntityNotExistsError:
			return result, nil
		default:
			s.logger.Info("error retrieving workflow", "err", err)
			return nil, ErrBatchStatusUnavailable
		}
	}
	if resp.WorkflowExecutionInfo == nil {
		s.logger.Info("error retrieving workflow execution details")
		return nil, ErrBatchStatusUnavailable
	}
	result.WorkflowID = resp.WorkflowExecutionInfo.Execution.WorkflowId
	result.RunID = resp.WorkflowExecutionInfo.Execution.RunId
	if resp.WorkflowExecutionInfo.CloseStatus != nil {
		var st = strings.ToLower(resp.WorkflowExecutionInfo.CloseStatus.String())
		result.Status = &st
		return result, nil
	}
	result.Running = true
	return result, nil
}

func (s *batchImpl) InitProcessingWorkflow(ctx context.Context, batchDir, key, pipelineName string) error {
	err := collection.InitProcessingWorkflow(ctx, s.cc, "", pipelineName, nil, false, key, batchDir, validation.Config{})
	if err != nil {
		s.logger.Error(err, "Error initializing processing workflow.")
	}
	return err
}
