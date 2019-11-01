package collection

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	goacollection "github.com/artefactual-labs/enduro/internal/api/gen/collection"
	"github.com/artefactual-labs/enduro/internal/cadence"

	"go.uber.org/cadence/.gen/go/shared"
)

// GoaWrapper returns a collectionImpl wrapper that implements
// goacollection.Service. It can handle types that are specific to the Goa API.
type goaWrapper struct {
	*collectionImpl
}

var _ goacollection.Service = (*goaWrapper)(nil)

// List all stored collections. It implements goacollection.Service.
func (w *goaWrapper) List(ctx context.Context, payload *goacollection.ListPayload) (goacollection.EnduroStoredCollectionCollection, error) {
	var query = "SELECT id, name, workflow_id, run_id, transfer_id, aip_id, original_id, status, CONVERT_TZ(created_at, @@session.time_zone, '+00:00') AS created_at, CONVERT_TZ(completed_at, @@session.time_zone, '+00:00') AS completed_at FROM collection"
	var args = []interface{}{}

	if payload.OriginalID != nil {
		query += " WHERE original_id = (?)"
		args = append(args, payload.OriginalID)
	}

	rows, err := w.db.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("error querying the database: %w", err)
	}
	defer rows.Close()

	var cols = []*goacollection.EnduroStoredCollection{}
	for rows.Next() {
		var c = Collection{}
		if err := rows.StructScan(&c); err != nil {
			return nil, fmt.Errorf("error scanning database result: %w", err)
		}
		cols = append(cols, c.Goa())
	}

	return cols, nil
}

// Show collection by ID. It implements goacollection.Service.
func (w *goaWrapper) Show(ctx context.Context, payload *goacollection.ShowPayload) (*goacollection.EnduroStoredCollection, error) {
	var query = "SELECT id, name, workflow_id, run_id, transfer_id, aip_id, original_id, status, CONVERT_TZ(created_at, @@session.time_zone, '+00:00') AS created_at, CONVERT_TZ(completed_at, @@session.time_zone, '+00:00') AS completed_at FROM collection WHERE id = (?)"
	var c = Collection{}

	query = w.db.Rebind(query)
	if err := w.db.GetContext(ctx, &c, query, payload.ID); err != nil {
		if err == sql.ErrNoRows {
			return nil, &goacollection.NotFound{ID: payload.ID}
		} else {
			return nil, err
		}
	}

	return c.Goa(), nil
}

// Delete collection by ID. It implements goacollection.Service.
//
// TODO: return error if it's still running?
func (w *goaWrapper) Delete(ctx context.Context, payload *goacollection.DeletePayload) error {
	var query = "DELETE FROM collection WHERE id = (?)"

	query = w.db.Rebind(query)
	res, err := w.db.ExecContext(ctx, query, payload.ID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return &goacollection.NotFound{ID: payload.ID}
	}

	return nil
}

// Cancel collection processing by ID. It implements goacollection.Service.
func (w *goaWrapper) Cancel(ctx context.Context, payload *goacollection.CancelPayload) error {
	var err error
	var goacol *goacollection.EnduroStoredCollection
	if goacol, err = w.Show(ctx, &goacollection.ShowPayload{ID: payload.ID}); err != nil {
		return err
	}

	if err := w.cc.CancelWorkflow(ctx, *goacol.WorkflowID, *goacol.RunID); err != nil {
		switch err.(type) {
		case *shared.InternalServiceError:
		case *shared.BadRequestError:
		case *shared.EntityNotExistsError:
			// TODO: return custom errors
		}
		return err
	}
	return nil
}

// Retry collection processing by ID. It implements goacollection.Service.
//
// TODO: collection and workflow packages belong to the same domain, they should live in the same package!
// TODO: conceptually Cadence workflows should handle retries, i.e. retry could be part of workflow code too (e.g. signals, children, etc).
// TODO: forbid retry when already running.
func (w *goaWrapper) Retry(ctx context.Context, payload *goacollection.RetryPayload) error {
	var err error
	var goacol *goacollection.EnduroStoredCollection
	if goacol, err = w.Show(ctx, &goacollection.ShowPayload{ID: payload.ID}); err != nil {
		return err
	}

	execution := &shared.WorkflowExecution{
		WorkflowId: goacol.WorkflowID,
		RunId:      goacol.RunID,
	}

	historyEvent, err := cadence.FirstHistoryEvent(ctx, w.cc, execution)
	if err != nil {
		return fmt.Errorf("error loading history of the previous workflow run: %w", err)
	}

	if historyEvent.GetEventType() != shared.EventTypeWorkflowExecutionStarted {
		return fmt.Errorf("error loading history of the previous workflow run: initiator state not found")
	}

	var input = historyEvent.WorkflowExecutionStartedEventAttributes.Input
	var attrs = bytes.Split(input, []byte("\n"))
	var req = &ProcessingWorkflowRequest{}

	if err := json.Unmarshal(attrs[0], req); err != nil {
		return fmt.Errorf("error loading state of the previous workflow run: %w", err)
	}

	req.WorkflowID = *goacol.WorkflowID
	req.CollectionID = goacol.ID
	if err := TriggerProcessingWorkflow(ctx, w.cc, req); err != nil {
		return fmt.Errorf("error triggering the new workflow instance: %w", err)
	}

	return nil
}

func (w *goaWrapper) Workflow(ctx context.Context, payload *goacollection.WorkflowPayload) (res *goacollection.EnduroCollectionWorkflowStatus, err error) {
	var goacol *goacollection.EnduroStoredCollection
	if goacol, err = w.Show(ctx, &goacollection.ShowPayload{ID: payload.ID}); err != nil {
		return nil, err
	}

	var resp = &goacollection.EnduroCollectionWorkflowStatus{
		History: []*goacollection.EnduroCollectionWorkflowHistory{},
	}

	we, err := w.cc.DescribeWorkflowExecution(ctx, *goacol.WorkflowID, *goacol.RunID)
	if err != nil {
		return nil, fmt.Errorf("error looking up workflow history: %v", err)
	}

	var status = "ACTIVE"
	if we.WorkflowExecutionInfo.CloseStatus != nil {
		status = we.WorkflowExecutionInfo.CloseStatus.String()
	}
	resp.Status = &status

	iter := w.cc.GetWorkflowHistory(ctx, *goacol.WorkflowID, *goacol.RunID, false, shared.HistoryEventFilterTypeAllEvent)
	for iter.HasNext() {
		event, err := iter.Next()
		if err != nil {
			return nil, fmt.Errorf("error looking up history events: %v", err)
		}

		var eventId uint = uint(*event.EventId)
		var eventType string = event.EventType.String()
		resp.History = append(resp.History, &goacollection.EnduroCollectionWorkflowHistory{
			ID:      &eventId,
			Type:    &eventType,
			Details: event,
		})
	}

	return resp, nil
}
