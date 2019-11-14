package collection

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	goacollection "github.com/artefactual-labs/enduro/internal/api/gen/collection"

	"github.com/jmoiron/sqlx"
	cadenceclient "go.uber.org/cadence/client"
)

type Service interface {
	Goa() goacollection.Service
	Create(context.Context, *Collection) error
	UpdateWorkflowStatus(ctx context.Context, ID uint, name string, workflowID, runID, transferID, aipID, pipelineID string, status Status, storedAt time.Time) error
}

type collectionImpl struct {
	db *sqlx.DB
	cc cadenceclient.Client
}

func NewService(db *sql.DB, cc cadenceclient.Client) *collectionImpl {
	return &collectionImpl{
		db: sqlx.NewDb(db, "mysql"),
		cc: cc,
	}
}

func (svc *collectionImpl) Goa() goacollection.Service {
	return &goaWrapper{
		collectionImpl: svc,
	}
}

func (svc *collectionImpl) Create(ctx context.Context, col *Collection) error {
	var query = `INSERT INTO collection (name, workflow_id, run_id, transfer_id, aip_id, original_id, pipeline_id, status) VALUES ((?), (?), (?), (?), (?), (?), (?), (?))`
	var args = []interface{}{
		col.Name,
		col.WorkflowID,
		col.RunID,
		col.TransferID,
		col.AIPID,
		col.OriginalID,
		col.PipelineID,
		col.Status,
	}

	query = svc.db.Rebind(query)
	res, err := svc.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("error inserting collection: %w", err)
	}

	var id int64
	if id, err = res.LastInsertId(); err != nil {
		return fmt.Errorf("error retrieving insert ID: %w", err)
	}

	col.ID = uint(id)

	return err
}

func (svc *collectionImpl) UpdateWorkflowStatus(ctx context.Context, ID uint, name string, workflowID, runID, transferID, aipID, pipelineID string, status Status, storedAt time.Time) error {
	// Ensure that storedAt is reset during retries.
	var completedAt = &storedAt
	if status == StatusInProgress {
		completedAt = nil
	}
	if completedAt != nil && completedAt.IsZero() {
		completedAt = nil
	}

	var query = `UPDATE collection SET name = (?), workflow_id = (?), run_id = (?), transfer_id = (?), aip_id = (?), pipeline_id = (?), status = (?), completed_at = (?) WHERE id = (?)`
	var args = []interface{}{
		name,
		workflowID,
		runID,
		transferID,
		aipID,
		pipelineID,
		status,
		completedAt,
		ID,
	}

	query = svc.db.Rebind(query)
	res, err := svc.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("error updating collection: %w", err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("error retrieving rows affected: %w", err)
	}
	if n != 1 {
		return fmt.Errorf("error updating collection: %d rows affected", n)
	}

	return nil
}
