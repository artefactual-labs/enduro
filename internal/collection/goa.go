package collection

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	cadencesdk_gen_shared "go.uber.org/cadence/.gen/go/shared"
	cadencesdk_client "go.uber.org/cadence/client"

	goacollection "github.com/artefactual-labs/enduro/internal/api/gen/collection"
	"github.com/artefactual-labs/enduro/internal/cadence"
)

var ErrBulkStatusUnavailable = errors.New("bulk status unavailable")

// GoaWrapper returns a collectionImpl wrapper that implements
// goacollection.Service. It can handle types that are specific to the Goa API.
type goaWrapper struct {
	*collectionImpl
}

var _ goacollection.Service = (*goaWrapper)(nil)

var patternMatchingCharReplacer = strings.NewReplacer(
	"%", "\\%",
	"_", "\\_",
)

// Monitor collection activity. It implements goacollection.Service.
func (w *goaWrapper) Monitor(ctx context.Context, stream goacollection.MonitorServerStream) error {
	defer stream.Close()

	// Subscribe to the event service.
	sub, err := w.events.Subscribe(ctx)
	if err != nil {
		return err
	}
	defer sub.Close()

	// Say hello to be nice.
	if err := stream.Send(&goacollection.EnduroMonitorUpdate{Type: "hello"}); err != nil {
		return err
	}

	// We'll use this ticker to ping the client once in a while to detect stale
	// connections. I'm not entirely sure this is needed, it may depend on the
	// client or the various middlewares.
	ticker := time.NewTicker(time.Second * 10)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil

		case <-ticker.C:
			if err := stream.Send(&goacollection.EnduroMonitorUpdate{Type: "ping"}); err != nil {
				return nil
			}

		case event, ok := <-sub.C():
			if !ok {
				return nil
			}

			if err := stream.Send(event); err != nil {
				return err
			}
		}
	}
}

// List all stored collections. It implements goacollection.Service.
func (w *goaWrapper) List(ctx context.Context, payload *goacollection.ListPayload) (*goacollection.ListResult, error) {
	query := "SELECT id, name, workflow_id, run_id, transfer_id, aip_id, original_id, pipeline_id, status, CONVERT_TZ(created_at, @@session.time_zone, '+00:00') AS created_at, CONVERT_TZ(started_at, @@session.time_zone, '+00:00') AS started_at, CONVERT_TZ(completed_at, @@session.time_zone, '+00:00') AS completed_at FROM collection"
	args := []interface{}{}

	// We extract one extra item so we can tell the next cursor.
	const limit = 20
	const limitSQL = "21"

	conds := [][2]string{}

	if payload.Name != nil {
		name := patternMatchingCharReplacer.Replace(*payload.Name) + "%"
		args = append(args, name)
		conds = append(conds, [2]string{"AND", "name LIKE (?)"})
	}
	if payload.OriginalID != nil {
		args = append(args, payload.OriginalID)
		conds = append(conds, [2]string{"AND", "original_id = (?)"})
	}
	if payload.TransferID != nil {
		args = append(args, payload.TransferID)
		conds = append(conds, [2]string{"AND", "transfer_id = (?)"})
	}
	if payload.AipID != nil {
		args = append(args, payload.AipID)
		conds = append(conds, [2]string{"AND", "aip_id = (?)"})
	}
	if payload.PipelineID != nil {
		args = append(args, payload.PipelineID)
		conds = append(conds, [2]string{"AND", "pipeline_id = (?)"})
	}
	if payload.Status != nil {
		args = append(args, NewStatus(*payload.Status))
		conds = append(conds, [2]string{"AND", "status = (?)"})
	}
	if payload.EarliestCreatedTime != nil {
		args = append(args, payload.EarliestCreatedTime)
		conds = append(conds, [2]string{"AND", "created_at >= (?)"})
	}
	if payload.LatestCreatedTime != nil {
		args = append(args, payload.LatestCreatedTime)
		conds = append(conds, [2]string{"AND", "created_at <= (?)"})
	}

	if payload.Cursor != nil {
		args = append(args, *payload.Cursor)
		conds = append(conds, [2]string{"AND", "id <= (?)"})
	}

	var where string
	for i, cond := range conds {
		if i == 0 {
			where = " WHERE " + cond[1]
			continue
		}
		where += fmt.Sprintf(" %s %s", cond[0], cond[1])
	}

	query += where + " ORDER BY id DESC LIMIT " + limitSQL

	query = w.db.Rebind(query)
	rows, err := w.db.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("error querying the database: %w", err)
	}
	defer rows.Close()

	cols := []*goacollection.EnduroStoredCollection{}
	for rows.Next() {
		c := Collection{}
		if err := rows.StructScan(&c); err != nil {
			return nil, fmt.Errorf("error scanning database result: %w", err)
		}
		cols = append(cols, c.Goa())
	}

	res := &goacollection.ListResult{
		Items: cols,
	}

	length := len(cols)
	if length > limit {
		last := cols[length-1]               // Capture last item.
		lastID := strconv.Itoa(int(last.ID)) // We also need its ID (cursor).
		res.Items = cols[:len(cols)-1]       // Remove it from the results.
		res.NextCursor = &lastID             // Populate cursor.
	}

	return res, nil
}

// Show collection by ID. It implements goacollection.Service.
func (w *goaWrapper) Show(ctx context.Context, payload *goacollection.ShowPayload) (*goacollection.EnduroStoredCollection, error) {
	c, err := w.read(ctx, payload.ID)
	if err == sql.ErrNoRows {
		return nil, &goacollection.CollectionNotfound{ID: payload.ID, Message: "not_found"}
	} else if err != nil {
		return nil, err
	}

	return c.Goa(), nil
}

// Delete collection by ID. It implements goacollection.Service.
//
// TODO: return error if it's still running?
func (w *goaWrapper) Delete(ctx context.Context, payload *goacollection.DeletePayload) error {
	query := "DELETE FROM collection WHERE id = (?)"

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
		return &goacollection.CollectionNotfound{ID: payload.ID, Message: "not_found"}
	}

	publishEvent(ctx, w.events, EventTypeCollectionDeleted, payload.ID)

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
		case *cadencesdk_gen_shared.InternalServiceError:
		case *cadencesdk_gen_shared.BadRequestError:
		case *cadencesdk_gen_shared.EntityNotExistsError:
		case *cadencesdk_gen_shared.WorkflowExecutionAlreadyCompletedError:
			// TODO: return custom errors
		}
		return err
	}

	publishEvent(ctx, w.events, EventTypeCollectionUpdated, payload.ID)

	return nil
}

// Retry collection processing by ID. It implements goacollection.Service.
//
// TODO: conceptually Cadence workflows should handle retries, i.e. retry could be part of workflow code too (e.g. signals, children, etc).
func (w *goaWrapper) Retry(ctx context.Context, payload *goacollection.RetryPayload) error {
	var err error
	var goacol *goacollection.EnduroStoredCollection
	if goacol, err = w.Show(ctx, &goacollection.ShowPayload{ID: payload.ID}); err != nil {
		return err
	}

	execution := &cadencesdk_gen_shared.WorkflowExecution{
		WorkflowId: goacol.WorkflowID,
		RunId:      goacol.RunID,
	}

	historyEvent, err := cadence.FirstHistoryEvent(ctx, w.cc, execution)
	if err != nil {
		return fmt.Errorf("error loading history of the previous workflow run: %w", err)
	}

	if historyEvent.GetEventType() != cadencesdk_gen_shared.EventTypeWorkflowExecutionStarted {
		return fmt.Errorf("error loading history of the previous workflow run: initiator state not found")
	}

	input := historyEvent.WorkflowExecutionStartedEventAttributes.Input
	attrs := bytes.Split(input, []byte("\n"))
	req := &ProcessingWorkflowRequest{}

	if err := json.Unmarshal(attrs[0], req); err != nil {
		return fmt.Errorf("error loading state of the previous workflow run: %w", err)
	}

	req.WorkflowID = *goacol.WorkflowID
	req.CollectionID = goacol.ID
	if err := InitProcessingWorkflow(ctx, w.cc, req); err != nil {
		return fmt.Errorf("error starting the new workflow instance: %w", err)
	}

	publishEvent(ctx, w.events, EventTypeCollectionUpdated, payload.ID)

	return nil
}

func (w *goaWrapper) Workflow(ctx context.Context, payload *goacollection.WorkflowPayload) (res *goacollection.EnduroCollectionWorkflowStatus, err error) {
	var goacol *goacollection.EnduroStoredCollection
	if goacol, err = w.Show(ctx, &goacollection.ShowPayload{ID: payload.ID}); err != nil {
		return nil, err
	}

	resp := &goacollection.EnduroCollectionWorkflowStatus{
		History: []*goacollection.EnduroCollectionWorkflowHistory{},
	}

	we, err := w.cc.DescribeWorkflowExecution(ctx, *goacol.WorkflowID, *goacol.RunID)
	if err != nil {
		switch err.(type) {
		case *cadencesdk_gen_shared.EntityNotExistsError:
			return nil, &goacollection.CollectionNotfound{Message: "not_found"}
		default:
			return nil, fmt.Errorf("error looking up history: %v", err)
		}
	}

	status := "ACTIVE"
	if we.WorkflowExecutionInfo.CloseStatus != nil {
		status = we.WorkflowExecutionInfo.CloseStatus.String()
	}
	resp.Status = &status

	iter := w.cc.GetWorkflowHistory(ctx, *goacol.WorkflowID, *goacol.RunID, false, cadencesdk_gen_shared.HistoryEventFilterTypeAllEvent)
	for iter.HasNext() {
		event, err := iter.Next()
		if err != nil {
			return nil, fmt.Errorf("error looking up history events: %v", err)
		}

		eventID := uint(*event.EventId)
		eventType := event.EventType.String()
		resp.History = append(resp.History, &goacollection.EnduroCollectionWorkflowHistory{
			ID:      &eventID,
			Type:    &eventType,
			Details: event,
		})
	}

	return resp, nil
}

// Download is a dummy implementation of goacollection.Service. The actual work
// (serving the file) is done from the API interceptor. Once goa supports this
// use case (it may never happen) we should be able to have a service method
// easy to adopt, e.g. with a io.Writer argument?
func (w *goaWrapper) Download(ctx context.Context, payload *goacollection.DownloadPayload) (res []byte, err error) {
	return []byte{}, nil
}

// Make decision for a pending collection by ID.
func (w *goaWrapper) Decide(ctx context.Context, payload *goacollection.DecidePayload) (err error) {
	c, err := w.read(ctx, payload.ID)

	if err == sql.ErrNoRows {
		return &goacollection.CollectionNotfound{ID: payload.ID, Message: "not_found"}
	} else if err != nil {
		return err
	}

	if c.DecisionToken == "" || c.Status != StatusPending {
		return goacollection.MakeNotValid(errors.New("collection is not awaiting decision"))
	}

	if payload.Option == "" {
		return goacollection.MakeNotValid(errors.New("missing decision option"))
	}

	if err := w.cc.CompleteActivity(ctx, []byte(c.DecisionToken), payload.Option, nil); err != nil {
		return err
	}

	publishEvent(ctx, w.events, EventTypeCollectionUpdated, payload.ID)

	return nil
}

func (w *goaWrapper) Bulk(ctx context.Context, payload *goacollection.BulkPayload) (*goacollection.BulkResult, error) {
	if payload.Size == 0 {
		return nil, goacollection.MakeNotValid(errors.New("size is zero"))
	}
	input := BulkWorkflowInput{
		Operation: BulkWorkflowOperation(payload.Operation),
		Status:    NewStatus(payload.Status),
		Size:      payload.Size,
	}

	opts := cadencesdk_client.StartWorkflowOptions{
		ID:                              BulkWorkflowID,
		WorkflowIDReusePolicy:           cadencesdk_client.WorkflowIDReusePolicyAllowDuplicate,
		TaskList:                        cadence.GlobalTaskListName,
		DecisionTaskStartToCloseTimeout: time.Second * 10,
		ExecutionStartToCloseTimeout:    time.Hour,
	}
	exec, err := w.cc.StartWorkflow(ctx, opts, BulkWorkflowName, input)
	if err != nil {
		switch err := err.(type) {
		case *cadencesdk_gen_shared.WorkflowExecutionAlreadyStartedError:
			return nil, goacollection.MakeNotAvailable(
				fmt.Errorf("error starting bulk - operation is already in progress (workflowID=%s runID=%s)",
					BulkWorkflowID, *err.RunId))
		default:
			w.logger.Info("error starting bulk", "err", err)
			return nil, fmt.Errorf("error starting bulk")
		}
	}

	return &goacollection.BulkResult{
		WorkflowID: exec.ID,
		RunID:      exec.RunID,
	}, nil
}

func (w *goaWrapper) BulkStatus(ctx context.Context) (*goacollection.BulkStatusResult, error) {
	result := &goacollection.BulkStatusResult{}

	resp, err := w.cc.DescribeWorkflowExecution(ctx, BulkWorkflowID, "")
	if err != nil {
		switch err := err.(type) {
		case *cadencesdk_gen_shared.EntityNotExistsError:
			// We've never seen a workflow run before.
			return result, nil
		default:
			w.logger.Info("error retrieving workflow", "err", err)
			return nil, ErrBulkStatusUnavailable
		}
	}

	if resp.WorkflowExecutionInfo == nil {
		w.logger.Info("error retrieving workflow execution details")
		return nil, ErrBulkStatusUnavailable
	}

	result.WorkflowID = resp.WorkflowExecutionInfo.Execution.WorkflowId
	result.RunID = resp.WorkflowExecutionInfo.Execution.RunId

	if resp.WorkflowExecutionInfo.StartTime != nil {
		t := time.Unix(0, *resp.WorkflowExecutionInfo.StartTime).Format(time.RFC3339)
		result.StartedAt = &t
	}

	if resp.WorkflowExecutionInfo.CloseTime != nil {
		t := time.Unix(0, *resp.WorkflowExecutionInfo.CloseTime).Format(time.RFC3339)
		result.ClosedAt = &t
	}

	// Workflow is not running!
	if resp.WorkflowExecutionInfo.CloseStatus != nil {
		st := strings.ToLower(resp.WorkflowExecutionInfo.CloseStatus.String())
		result.Status = &st

		return result, nil
	}

	result.Running = true

	// We can use the status property to communicate progress from heartbeats.
	length := len(resp.PendingActivities)
	if length > 0 {
		latest := resp.PendingActivities[length-1]
		progress := &BulkProgress{}
		if err := json.Unmarshal(latest.HeartbeatDetails, progress); err == nil {
			status := fmt.Sprintf("Processing collection %d (done: %d)", progress.CurrentID, progress.Count)
			result.Status = &status
		}
	}

	return result, nil
}
