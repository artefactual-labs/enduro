package collection

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
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

type execRecorderDB struct {
	db    *sql.DB
	query string
	args  []any
}

var execRecorderDriverID atomic.Uint64

func newExecRecorderDB(t *testing.T) *execRecorderDB {
	t.Helper()

	recorder := &execRecorderDB{}
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
	c.recorder.args = make([]any, len(args))
	for i, arg := range args {
		c.recorder.args[i] = arg.Value
	}

	return driver.RowsAffected(1), nil
}
