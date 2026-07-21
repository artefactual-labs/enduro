package workflow

import (
	"errors"
	"fmt"
	"time"

	temporalsdk_temporal "go.temporal.io/sdk/temporal"
	temporalsdk_workflow "go.temporal.io/sdk/workflow"

	"github.com/artefactual-labs/enduro/internal/collection"
)

const operatorDecisionTimeout = 7 * 24 * time.Hour

// ErrOperatorDecisionAbandoned indicates that an operator abandoned processing.
var ErrOperatorDecisionAbandoned = errors.New("user abandoned")

type operatorDecisionHandler struct {
	ctx      temporalsdk_workflow.Context
	awaiting bool
	decision collection.ProcessingWorkflowDecision
}

func newOperatorDecisionHandler(ctx temporalsdk_workflow.Context) (*operatorDecisionHandler, error) {
	h := &operatorDecisionHandler{ctx: ctx}
	err := temporalsdk_workflow.SetUpdateHandlerWithOptions(
		ctx,
		collection.ProcessingWorkflowDecisionUpdateName,
		func(_ temporalsdk_workflow.Context, decision collection.ProcessingWorkflowDecision) error {
			h.decision = decision
			h.awaiting = false
			return nil
		},
		temporalsdk_workflow.UpdateHandlerOptions{
			Validator: func(decision collection.ProcessingWorkflowDecision) error {
				if _, err := collection.ParseProcessingWorkflowDecision(string(decision)); err != nil {
					return err
				}
				if !h.awaiting {
					return errors.New("workflow is not awaiting an operator decision")
				}
				return nil
			},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("error registering operator decision handler: %w", err)
	}

	return h, nil
}

func (h *operatorDecisionHandler) await(
	ctx temporalsdk_workflow.Context,
	colsvc collection.Service,
	colID uint,
) (collection.ProcessingWorkflowDecision, error) {
	h.awaiting = true
	h.decision = ""
	defer func() { h.awaiting = false }()

	activityOpts := withLocalActivityOpts(h.ctx)
	err := temporalsdk_workflow.ExecuteLocalActivity(
		activityOpts,
		setStatusLocalActivity,
		colsvc,
		colID,
		collection.StatusPending,
	).Get(activityOpts, nil)
	if err != nil {
		return "", fmt.Errorf("error setting collection status to pending: %w", err)
	}

	ok, err := temporalsdk_workflow.AwaitWithTimeout(ctx, operatorDecisionTimeout, func() bool {
		return h.decision != ""
	})
	if err != nil {
		return "", err
	}
	if !ok {
		return collection.ProcessingWorkflowDecisionAbandon, nil
	}

	return h.decision, nil
}

func executeActivityWithOperatorDecision(
	ctx temporalsdk_workflow.Context,
	decisions *operatorDecisionHandler,
	colsvc collection.Service,
	colID uint,
	opts temporalsdk_workflow.ActivityOptions,
	act any,
	args ...any,
) error {
	decision := collection.ProcessingWorkflowDecisionRetry

	for {
		activityOptions := opts
		if decision == collection.ProcessingWorkflowDecisionRetryOnce {
			activityOptions.RetryPolicy = &temporalsdk_temporal.RetryPolicy{MaximumAttempts: 1}
		}
		activityCtx := temporalsdk_workflow.WithActivityOptions(ctx, activityOptions)
		err := temporalsdk_workflow.ExecuteActivity(activityCtx, act, args...).Get(activityCtx, nil)
		if err == nil {
			return nil
		}
		if !requiresOperatorDecision(err) {
			return err
		}

		decision, err = decisions.await(ctx, colsvc, colID)
		if err != nil {
			return err
		}

		switch decision {
		case collection.ProcessingWorkflowDecisionRetry,
			collection.ProcessingWorkflowDecisionRetryOnce:
			statusOpts := withLocalActivityOpts(decisions.ctx)
			if err := temporalsdk_workflow.ExecuteLocalActivity(
				statusOpts,
				setStatusInProgressLocalActivity,
				colsvc,
				colID,
				time.Time{},
			).Get(statusOpts, nil); err != nil {
				return fmt.Errorf("error setting collection status to in progress: %w", err)
			}
			continue
		case collection.ProcessingWorkflowDecisionAbandon:
			return ErrOperatorDecisionAbandoned
		default:
			return fmt.Errorf("received unknown operator decision %q", decision)
		}
	}
}

func requiresOperatorDecision(err error) bool {
	return !errors.Is(err, temporalsdk_workflow.ErrSessionFailed) &&
		!temporalsdk_temporal.IsCanceledError(err)
}
