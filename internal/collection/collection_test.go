package collection

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
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
		svc := NewService(testLogger(), recorder.db, nil, "", nil)
		startedAt := time.Date(2026, time.June, 24, 8, 30, 0, 0, time.UTC)

		err := svc.SetStatusInProgress(context.Background(), 42, startedAt)

		assert.NilError(t, err)
		assert.Equal(t, recorder.query, "UPDATE collection SET status = (?), started_at = COALESCE(started_at, (?)) WHERE id = (?)")
		assert.DeepEqual(t, recorder.args, []any{
			int64(StatusInProgress),
			startedAt,
			int64(42),
		})
	})

	t.Run("Updates only the status without a started_at value", func(t *testing.T) {
		t.Parallel()

		recorder := newExecRecorderDB(t)
		svc := NewService(testLogger(), recorder.db, nil, "", nil)

		err := svc.SetStatusInProgress(context.Background(), 42, time.Time{})

		assert.NilError(t, err)
		assert.Equal(t, recorder.query, "UPDATE collection SET status = (?) WHERE id = (?)")
		assert.DeepEqual(t, recorder.args, []any{
			int64(StatusInProgress),
			int64(42),
		})
	})
}

type execRecorderDB struct {
	db *sql.DB

	query string
	args  []any

	execQuery string
	execArgs  []any
	querySQL  string
	queryArgs []any

	rowsAffected int64
	execErr      error
	queryErr     error
	row          *Collection
}

var execRecorderDriverID atomic.Uint64

func newExecRecorderDB(t *testing.T) *execRecorderDB {
	t.Helper()

	recorder := &execRecorderDB{rowsAffected: 1}
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
func (c execRecorderConn) Begin() (driver.Tx, error)           { return nil, driver.ErrSkip }

func (c execRecorderConn) ExecContext(_ context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	c.recorder.query = query
	c.recorder.execQuery = query
	c.recorder.args = make([]any, len(args))
	c.recorder.execArgs = make([]any, len(args))
	for i, arg := range args {
		c.recorder.args[i] = arg.Value
		c.recorder.execArgs[i] = arg.Value
	}

	if c.recorder.execErr != nil {
		return nil, c.recorder.execErr
	}

	return driver.RowsAffected(c.recorder.rowsAffected), nil
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

	return &collectionRows{row: c.recorder.row}, nil
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
		"decision_token",
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
		r.row.DecisionToken,
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
