package cadence

import (
	"context"
	"errors"
	"fmt"

	cadencesdk_gen_shared "go.uber.org/cadence/.gen/go/shared"
	cadencesdk_client "go.uber.org/cadence/client"
)

func HistoryEvents(ctx context.Context, cc cadencesdk_client.Client, exec *cadencesdk_gen_shared.WorkflowExecution, poll bool) ([]*cadencesdk_gen_shared.HistoryEvent, error) {
	iter := cc.GetWorkflowHistory(ctx, exec.GetWorkflowId(), exec.GetRunId(), poll, cadencesdk_gen_shared.HistoryEventFilterTypeAllEvent)
	var events []*cadencesdk_gen_shared.HistoryEvent
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

func FirstHistoryEvent(ctx context.Context, cc cadencesdk_client.Client, exec *cadencesdk_gen_shared.WorkflowExecution) (event *cadencesdk_gen_shared.HistoryEvent, err error) {
	const polling = false
	iter := cc.GetWorkflowHistory(ctx, exec.GetWorkflowId(), exec.GetRunId(), polling, cadencesdk_gen_shared.HistoryEventFilterTypeAllEvent)

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
