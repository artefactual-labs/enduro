package batch

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-logr/logr"
	cadencesdk_gen_shared "go.uber.org/cadence/.gen/go/shared"
	cadencesdk_client "go.uber.org/cadence/client"

	goabatch "github.com/artefactual-labs/enduro/internal/api/gen/batch"
	"github.com/artefactual-labs/enduro/internal/cadence"
	"github.com/artefactual-labs/enduro/internal/collection"
	"github.com/artefactual-labs/enduro/internal/validation"
)

var ErrBatchStatusUnavailable = errors.New("batch status unavailable")

type Service interface {
	Submit(context.Context, *goabatch.SubmitPayload) (res *goabatch.BatchResult, err error)
	Status(context.Context) (res *goabatch.BatchStatusResult, err error)
	Hints(context.Context) (res *goabatch.BatchHintsResult, err error)
	InitProcessingWorkflow(ctx context.Context, req *collection.ProcessingWorkflowRequest) error
}

type batchImpl struct {
	logger logr.Logger
	cc     cadencesdk_client.Client

	// A list of completedDirs reported by the watcher configuration. This is
	// used to provide the user with possible known values.
	completedDirs []string
}

var _ Service = (*batchImpl)(nil)

func NewService(logger logr.Logger, cc cadencesdk_client.Client, completedDirs []string) *batchImpl {
	return &batchImpl{
		logger:        logger,
		cc:            cc,
		completedDirs: completedDirs,
	}
}

func (s *batchImpl) Submit(ctx context.Context, payload *goabatch.SubmitPayload) (*goabatch.BatchResult, error) {
	if payload.Path == "" {
		return nil, goabatch.MakeNotValid(errors.New("error starting batch - path is empty"))
	}
	input := BatchWorkflowInput{
		Path: payload.Path,
	}
	if payload.Pipeline != nil {
		input.PipelineName = *payload.Pipeline
	}
	if payload.ProcessingConfig != nil {
		input.ProcessingConfig = *payload.ProcessingConfig
	}
	if payload.CompletedDir != nil {
		input.CompletedDir = *payload.CompletedDir
	}
	if payload.RetentionPeriod != nil {
		dur, err := time.ParseDuration(*payload.RetentionPeriod)
		if err != nil {
			return nil, goabatch.MakeNotValid(errors.New("error starting batch - retention period format is invalid"))
		}
		input.RetentionPeriod = &dur
	}
	opts := cadencesdk_client.StartWorkflowOptions{
		ID:                              BatchWorkflowID,
		WorkflowIDReusePolicy:           cadencesdk_client.WorkflowIDReusePolicyAllowDuplicate,
		TaskList:                        cadence.GlobalTaskListName,
		DecisionTaskStartToCloseTimeout: time.Second * 10,
		ExecutionStartToCloseTimeout:    time.Hour,
	}
	exec, err := s.cc.StartWorkflow(ctx, opts, BatchWorkflowName, input)
	if err != nil {
		switch err := err.(type) {
		case *cadencesdk_gen_shared.WorkflowExecutionAlreadyStartedError:
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
		case *cadencesdk_gen_shared.EntityNotExistsError:
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
		st := strings.ToLower(resp.WorkflowExecutionInfo.CloseStatus.String())
		result.Status = &st
		return result, nil
	}
	result.Running = true
	return result, nil
}

func (s *batchImpl) Hints(ctx context.Context) (*goabatch.BatchHintsResult, error) {
	result := &goabatch.BatchHintsResult{
		CompletedDirs: s.completedDirs,
	}
	return result, nil
}

func (s *batchImpl) InitProcessingWorkflow(ctx context.Context, req *collection.ProcessingWorkflowRequest) error {
	req.ValidationConfig = validation.Config{}
	err := collection.InitProcessingWorkflow(ctx, s.cc, req)
	if err != nil {
		s.logger.Error(err, "Error initializing processing workflow.")
	}
	return err
}
