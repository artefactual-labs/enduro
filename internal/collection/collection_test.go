package collection

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"gotest.tools/v3/assert"
)

func TestUpdateReconciliationState(t *testing.T) {
	t.Parallel()

	t.Run("Writes provided reconciliation values", func(t *testing.T) {
		t.Parallel()

		recorder := newExecRecorderDB(t)
		svc := NewService(testLogger(), recorder.db, nil, "", nil)

		aipStoredAt := time.Date(2026, time.March, 18, 8, 0, 0, 0, time.UTC)
		checkedAt := aipStoredAt.Add(5 * time.Minute)
		status := "complete"
		errMsg := "replica lag"

		err := svc.UpdateReconciliationState(context.Background(), 42, &aipStoredAt, &checkedAt, &status, &errMsg)

		assert.NilError(t, err)
		assert.Equal(t, recorder.query, "UPDATE collection SET aip_stored_at = (?), reconciliation_checked_at = (?), reconciliation_status = (?), reconciliation_error = (?) WHERE id = (?)")
		assert.Equal(t, len(recorder.args), 5)
		assert.DeepEqual(t, recorder.args[0], aipStoredAt)
		assert.DeepEqual(t, recorder.args[1], checkedAt)
		assert.Equal(t, recorder.args[2], status)
		assert.Equal(t, recorder.args[3], errMsg)
		assert.Equal(t, recorder.args[4], int64(42))
	})

	t.Run("Clears reconciliation values when nil", func(t *testing.T) {
		t.Parallel()

		recorder := newExecRecorderDB(t)
		svc := NewService(testLogger(), recorder.db, nil, "", nil)

		err := svc.UpdateReconciliationState(context.Background(), 42, nil, nil, nil, nil)

		assert.NilError(t, err)
		assert.Equal(t, len(recorder.args), 5)
		assert.Assert(t, recorder.args[0] == nil)
		assert.Assert(t, recorder.args[1] == nil)
		assert.Assert(t, recorder.args[2] == nil)
		assert.Assert(t, recorder.args[3] == nil)
		assert.Equal(t, recorder.args[4], int64(42))
	})
}

func TestSetStatusInProgress(t *testing.T) {
	t.Parallel()

	t.Run("Preserves an existing started_at value", func(t *testing.T) {
		t.Parallel()

		recorder := newExecRecorderDB(t)
		recorder.row = &Collection{WorkflowID: "workflow-42", RunID: "run-42", Status: StatusQueued}
		svc := NewService(testLogger(), recorder.db, nil, "", nil)
		startedAt := time.Date(2026, time.June, 24, 8, 30, 0, 0, time.UTC)

		err := svc.SetStatusInProgress(context.Background(), 42, startedAt)

		assert.NilError(t, err)
		assert.Equal(t, recorder.execQueries[0], "UPDATE collection SET status = (?), started_at = COALESCE(started_at, (?)) WHERE id = (?)")
		assert.DeepEqual(t, recorder.execArgsList[0], []any{
			int64(StatusInProgress),
			startedAt,
			int64(42),
		})
		assert.Equal(t, recorder.execQueries[1], "INSERT INTO collection_status_transition (collection_id, workflow_id, run_id, previous_status, status, is_run_start, reason) VALUES ((?), (?), (?), (?), (?), (?), (?))")
		assert.DeepEqual(t, recorder.execArgsList[1], []any{
			int64(42),
			"workflow-42",
			"run-42",
			int64(StatusQueued),
			int64(StatusInProgress),
			false,
			"pipeline_acquired",
		})
		assert.Assert(t, recorder.committed)
	})

	t.Run("Updates only the status without a started_at value", func(t *testing.T) {
		t.Parallel()

		recorder := newExecRecorderDB(t)
		recorder.row = &Collection{WorkflowID: "workflow-42", RunID: "run-42", Status: StatusPending}
		svc := NewService(testLogger(), recorder.db, nil, "", nil)

		err := svc.SetStatusInProgress(context.Background(), 42, time.Time{})

		assert.NilError(t, err)
		assert.Equal(t, recorder.execQueries[0], "UPDATE collection SET status = (?) WHERE id = (?)")
		assert.DeepEqual(t, recorder.execArgsList[0], []any{
			int64(StatusInProgress),
			int64(42),
		})
		assert.Equal(t, recorder.execArgsList[1][6], "operator_decision_received")
		assert.Assert(t, recorder.committed)
	})
}

func TestSetStatusDoesNotDuplicateTransition(t *testing.T) {
	t.Parallel()

	recorder := newExecRecorderDB(t)
	recorder.row = &Collection{WorkflowID: "workflow-42", RunID: "run-42", Status: StatusPending}
	svc := NewService(testLogger(), recorder.db, nil, "", nil)

	err := svc.SetStatus(context.Background(), 42, StatusPending)

	assert.NilError(t, err)
	assert.Equal(t, len(recorder.execQueries), 1)
	assert.Assert(t, recorder.committed)
}

func TestSetStatusRecordsOperatorDecision(t *testing.T) {
	t.Parallel()

	recorder := newExecRecorderDB(t)
	recorder.row = &Collection{WorkflowID: "workflow-42", RunID: "run-42", Status: StatusInProgress}
	svc := NewService(testLogger(), recorder.db, nil, "", nil)

	err := svc.SetStatus(context.Background(), 42, StatusPending)

	assert.NilError(t, err)
	assert.DeepEqual(t, recorder.execArgsList[1], []any{
		int64(42),
		"workflow-42",
		"run-42",
		int64(StatusInProgress),
		int64(StatusPending),
		false,
		"operator_decision_required",
	})
}

func TestCreateRecordsRunStartTransition(t *testing.T) {
	t.Parallel()

	recorder := newExecRecorderDB(t)
	svc := NewService(testLogger(), recorder.db, nil, "", nil)
	col := &Collection{
		Name:       "collection",
		WorkflowID: "workflow-42",
		RunID:      "run-42",
		Status:     StatusQueued,
	}

	err := svc.Create(context.Background(), col)

	assert.NilError(t, err)
	assert.Equal(t, col.ID, uint(42))
	assert.Equal(t, len(recorder.execQueries), 2)
	assert.DeepEqual(t, recorder.execArgsList[1], []any{
		int64(42),
		"workflow-42",
		"run-42",
		nil,
		int64(StatusQueued),
		true,
		"collection_created",
	})
	assert.Assert(t, recorder.committed)
}

func TestStatusTransitionFailureRollsBack(t *testing.T) {
	t.Parallel()

	recorder := newExecRecorderDB(t)
	recorder.row = &Collection{WorkflowID: "workflow-42", RunID: "run-42", Status: StatusQueued}
	recorder.execErr = errTestDB
	recorder.execErrAt = 2
	svc := NewService(testLogger(), recorder.db, nil, "", nil)

	err := svc.SetStatus(context.Background(), 42, StatusError)

	assert.ErrorContains(t, err, "error inserting collection status transition")
	assert.Assert(t, recorder.rolledBack)
	assert.Assert(t, !recorder.committed)
}

func TestUpdateWorkflowStatusRecordsRetryRunStart(t *testing.T) {
	t.Parallel()

	recorder := newExecRecorderDB(t)
	recorder.row = &Collection{WorkflowID: "workflow-42", RunID: "old-run", Status: StatusError}
	svc := NewService(testLogger(), recorder.db, nil, "", nil)

	err := svc.UpdateWorkflowStatus(
		context.Background(),
		42,
		"collection",
		"workflow-42",
		"new-run",
		"",
		"",
		"",
		StatusQueued,
		time.Time{},
	)

	assert.NilError(t, err)
	assert.DeepEqual(t, recorder.execArgsList[1], []any{
		int64(42),
		"workflow-42",
		"new-run",
		int64(StatusError),
		int64(StatusQueued),
		true,
		"workflow_retried",
	})
}

func TestStatusHistoryAvailability(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		transitions []StatusTransition
		runID       string
		want        string
	}{
		"unavailable without transitions": {
			runID: "run-42",
			want:  StatusHistoryUnavailable,
		},
		"partial when recording started during run": {
			transitions: []StatusTransition{{RunID: "run-42"}},
			runID:       "run-42",
			want:        StatusHistoryPartial,
		},
		"available from run start": {
			transitions: []StatusTransition{{RunID: "run-42", IsRunStart: true}},
			runID:       "run-42",
			want:        StatusHistoryAvailable,
		},
		"uses the current run": {
			transitions: []StatusTransition{
				{RunID: "old-run", IsRunStart: true},
				{RunID: "run-42"},
			},
			runID: "run-42",
			want:  StatusHistoryPartial,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, statusHistoryAvailability(tc.transitions, tc.runID), tc.want)
		})
	}
}

func TestCheckDuplicateIgnoresFailedAndAbandonedCollections(t *testing.T) {
	t.Parallel()

	duplicateExists := false
	recorder := newExecRecorderDB(t)
	recorder.queryBool = &duplicateExists
	svc := NewService(testLogger(), recorder.db, nil, "", nil)

	got, err := svc.CheckDuplicate(context.Background(), 42)

	assert.NilError(t, err)
	assert.Equal(t, got, false)
	assert.Equal(t, recorder.querySQL, "SELECT EXISTS(SELECT 1 FROM collection c1 WHERE c1.name = (SELECT name FROM collection WHERE id = ?) AND c1.id <> ? AND c1.status NOT IN (?, ?))")
	assert.DeepEqual(t, recorder.queryArgs, []any{
		int64(42),
		int64(42),
		int64(StatusError),
		int64(StatusAbandoned),
	})
}

type execRecorderDB struct {
	db *sql.DB

	query string
	args  []any

	execQuery    string
	execArgs     []any
	execQueries  []string
	execArgsList [][]any
	querySQL     string
	queryArgs    []any

	rowsAffected int64
	execErr      error
	execErrAt    int
	queryErr     error
	row          *Collection
	queryBool    *bool
	transitions  []StatusTransition
	lastInsertID int64
	committed    bool
	rolledBack   bool
}

var execRecorderDriverID atomic.Uint64

func newExecRecorderDB(t *testing.T) *execRecorderDB {
	t.Helper()

	recorder := &execRecorderDB{rowsAffected: 1, lastInsertID: 42}
	driverName := fmt.Sprintf("collection-test-driver-%d", execRecorderDriverID.Add(1))
	sql.Register(driverName, execRecorderDriver{recorder: recorder})

	db, err := sql.Open(driverName, "")
	assert.NilError(t, err)
	t.Cleanup(func() {
		_ = db.Close()
	})

	recorder.db = db

	return recorder
}

func testLogger() logr.Logger {
	return logr.Discard()
}

type execRecorderDriver struct {
	recorder *execRecorderDB
}

func (d execRecorderDriver) Open(string) (driver.Conn, error) {
	return execRecorderConn(d), nil
}

type execRecorderConn struct {
	recorder *execRecorderDB
}

func (c execRecorderConn) Prepare(string) (driver.Stmt, error) { return nil, driver.ErrSkip }
func (c execRecorderConn) Close() error                        { return nil }
func (c execRecorderConn) Begin() (driver.Tx, error) {
	return &execRecorderTx{recorder: c.recorder}, nil
}

func (c execRecorderConn) BeginTx(context.Context, driver.TxOptions) (driver.Tx, error) {
	return c.Begin()
}

func (c execRecorderConn) ExecContext(_ context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	c.recorder.query = query
	c.recorder.execQuery = query
	c.recorder.execQueries = append(c.recorder.execQueries, query)
	c.recorder.args = make([]any, len(args))
	c.recorder.execArgs = make([]any, len(args))
	for i, arg := range args {
		c.recorder.args[i] = arg.Value
		c.recorder.execArgs[i] = arg.Value
	}
	c.recorder.execArgsList = append(c.recorder.execArgsList, append([]any(nil), c.recorder.execArgs...))

	if c.recorder.execErr != nil && (c.recorder.execErrAt == 0 || c.recorder.execErrAt == len(c.recorder.execQueries)) {
		return nil, c.recorder.execErr
	}

	return execRecorderResult{lastInsertID: c.recorder.lastInsertID, rowsAffected: c.recorder.rowsAffected}, nil
}

func (c execRecorderConn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	c.recorder.querySQL = query
	c.recorder.queryArgs = make([]any, len(args))
	for i, arg := range args {
		c.recorder.queryArgs[i] = arg.Value
	}

	if c.recorder.queryErr != nil {
		return nil, c.recorder.queryErr
	}
	if c.recorder.queryBool != nil {
		return &boolRows{value: *c.recorder.queryBool}, nil
	}
	if strings.Contains(query, "FROM collection_status_transition") {
		return &statusTransitionRows{transitions: c.recorder.transitions}, nil
	}
	if strings.Contains(query, "SELECT workflow_id, run_id, status FROM collection") {
		return &collectionStatusRows{row: c.recorder.row}, nil
	}

	return &collectionRows{row: c.recorder.row}, nil
}

type execRecorderTx struct {
	recorder *execRecorderDB
}

func (tx *execRecorderTx) Commit() error {
	tx.recorder.committed = true
	return nil
}

func (tx *execRecorderTx) Rollback() error {
	tx.recorder.rolledBack = true
	return nil
}

type execRecorderResult struct {
	lastInsertID int64
	rowsAffected int64
}

func (r execRecorderResult) LastInsertId() (int64, error) { return r.lastInsertID, nil }
func (r execRecorderResult) RowsAffected() (int64, error) { return r.rowsAffected, nil }

type collectionStatusRows struct {
	row  *Collection
	done bool
}

func (r *collectionStatusRows) Columns() []string {
	return []string{"workflow_id", "run_id", "status"}
}

func (r *collectionStatusRows) Close() error { return nil }

func (r *collectionStatusRows) Next(dest []driver.Value) error {
	if r.done || r.row == nil {
		return io.EOF
	}
	r.done = true
	dest[0] = r.row.WorkflowID
	dest[1] = r.row.RunID
	dest[2] = int64(r.row.Status)
	return nil
}

type statusTransitionRows struct {
	transitions []StatusTransition
	index       int
}

func (r *statusTransitionRows) Columns() []string {
	return []string{"id", "collection_id", "workflow_id", "run_id", "previous_status", "status", "occurred_at", "is_run_start", "reason"}
}

func (r *statusTransitionRows) Close() error { return nil }

func (r *statusTransitionRows) Next(dest []driver.Value) error {
	if r.index >= len(r.transitions) {
		return io.EOF
	}
	transition := r.transitions[r.index]
	r.index++
	dest[0] = int64(transition.ID)
	dest[1] = int64(transition.CollectionID)
	dest[2] = transition.WorkflowID
	dest[3] = transition.RunID
	if transition.PreviousStatus.Valid {
		dest[4] = transition.PreviousStatus.Int64
	}
	dest[5] = int64(transition.Status)
	dest[6] = transition.OccurredAt
	dest[7] = transition.IsRunStart
	if transition.Reason.Valid {
		dest[8] = transition.Reason.String
	}
	return nil
}

type boolRows struct {
	value bool
	done  bool
}

func (r *boolRows) Columns() []string {
	return []string{"exists"}
}

func (r *boolRows) Close() error {
	return nil
}

func (r *boolRows) Next(dest []driver.Value) error {
	if r.done {
		return io.EOF
	}
	r.done = true
	dest[0] = r.value

	return nil
}

type collectionRows struct {
	row  *Collection
	done bool
}

func (r *collectionRows) Columns() []string {
	return []string{
		"id",
		"name",
		"workflow_id",
		"run_id",
		"transfer_id",
		"aip_id",
		"original_id",
		"pipeline_id",
		"status",
		"created_at",
		"started_at",
		"completed_at",
		"aip_stored_at",
		"reconciliation_status",
		"reconciliation_checked_at",
		"reconciliation_error",
	}
}

func (r *collectionRows) Close() error {
	return nil
}

func (r *collectionRows) Next(dest []driver.Value) error {
	if r.done || r.row == nil {
		return io.EOF
	}
	r.done = true

	values := []driver.Value{
		int64(r.row.ID),
		r.row.Name,
		r.row.WorkflowID,
		r.row.RunID,
		r.row.TransferID,
		r.row.AIPID,
		r.row.OriginalID,
		r.row.PipelineID,
		int64(r.row.Status),
		r.row.CreatedAt,
		nullTimeValue(r.row.StartedAt),
		nullTimeValue(r.row.CompletedAt),
		nullTimeValue(r.row.AIPStoredAt),
		nullStringValue(r.row.ReconciliationStatus),
		nullTimeValue(r.row.ReconciliationCheckedAt),
		nullStringValue(r.row.ReconciliationError),
	}
	copy(dest, values)

	return nil
}

func nullTimeValue(v sql.NullTime) driver.Value {
	if !v.Valid {
		return nil
	}
	return v.Time
}

func nullStringValue(v sql.NullString) driver.Value {
	if !v.Valid {
		return nil
	}
	return v.String
}

var errTestDB = errors.New("database error")
