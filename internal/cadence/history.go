package cadence

import (
	"context"
	"errors"
	"fmt"

	"go.uber.org/cadence/.gen/go/shared"
	"go.uber.org/cadence/client"
)

func HistoryEvents(ctx context.Context, cc client.Client, exec *shared.WorkflowExecution, poll bool) ([]*shared.HistoryEvent, error) {
	iter := cc.GetWorkflowHistory(ctx, exec.GetWorkflowId(), exec.GetRunId(), poll, shared.HistoryEventFilterTypeAllEvent)
	var events []*shared.HistoryEvent
	for iter.HasNext() {
		event, err := iter.Next()
		if err != nil {
			return nil, fmt.Errorf("error looking up history events: %w", err)
		}

		events = append(events, event)
	}

	if len(events) == 0 {
		return nil, errors.New("error looking up history events: history is empty")
	}

	return events, nil
}

func FirstHistoryEvent(ctx context.Context, cc client.Client, exec *shared.WorkflowExecution) (event *shared.HistoryEvent, err error) {
	const polling = false
	iter := cc.GetWorkflowHistory(ctx, exec.GetWorkflowId(), exec.GetRunId(), polling, shared.HistoryEventFilterTypeAllEvent)

	for iter.HasNext() {
		event, err = iter.Next()
		if err != nil {
			return nil, fmt.Errorf("error looking up history events: %w", err)
		} else {
			break
		}
	}

	return event, nil
}
