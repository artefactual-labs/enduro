package temporal

import (
	"context"
	"errors"
	"fmt"

	temporalapi_common "go.temporal.io/api/common/v1"
	temporalapi_enums "go.temporal.io/api/enums/v1"
	temporalapi_history "go.temporal.io/api/history/v1"
	temporalsdk_client "go.temporal.io/sdk/client"
)

func HistoryEvents(ctx context.Context, cc temporalsdk_client.Client, exec *temporalapi_common.WorkflowExecution, poll bool) ([]*temporalapi_history.HistoryEvent, error) {
	iter := cc.GetWorkflowHistory(ctx, exec.GetWorkflowId(), exec.GetRunId(), poll, temporalapi_enums.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT)
	var events []*temporalapi_history.HistoryEvent
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

func FirstHistoryEvent(ctx context.Context, cc temporalsdk_client.Client, exec *temporalapi_common.WorkflowExecution) (event *temporalapi_history.HistoryEvent, err error) {
	const polling = false
	iter := cc.GetWorkflowHistory(ctx, exec.GetWorkflowId(), exec.GetRunId(), polling, temporalapi_enums.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT)

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
