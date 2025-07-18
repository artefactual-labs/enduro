package collection

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/jmoiron/sqlx"
	temporalsdk_client "go.temporal.io/sdk/client"

	goacollection "github.com/artefactual-labs/enduro/internal/api/gen/collection"
	"github.com/artefactual-labs/enduro/internal/pipeline"
)

type Service interface {
	// Goa returns an implementation of the goacollection Service.
	Goa() goacollection.Service
	Create(context.Context, *Collection) error
	CheckDuplicate(ctx context.Context, id uint) (bool, error)
	UpdateWorkflowStatus(ctx context.Context, ID uint, name, workflowID, runID, transferID, aipID, pipelineID string, status Status, storedAt time.Time) error
	SetStatus(ctx context.Context, ID uint, status Status) error
	SetStatusInProgress(ctx context.Context, ID uint, startedAt time.Time) error
	SetStatusPending(ctx context.Context, ID uint, taskToken []byte) error
	SetOriginalID(ctx context.Context, ID uint, originalID string) error
}

type collectionImpl struct {
	logger    logr.Logger
	db        *sqlx.DB
	cc        temporalsdk_client.Client
	taskQueue string

	registry *pipeline.Registry

	// Destination for events to be published.
	events EventService
}

var _ Service = (*collectionImpl)(nil)

func NewService(logger logr.Logger, db *sql.DB, cc temporalsdk_client.Client, taskQueue string, registry *pipeline.Registry) *collectionImpl {
	return &collectionImpl{
		logger:    logger,
		db:        sqlx.NewDb(db, "mysql"),
		cc:        cc,
		taskQueue: taskQueue,
		registry:  registry,
		events:    NewEventService(),
	}
}

func (svc *collectionImpl) Goa() goacollection.Service {
	return &goaWrapper{
		collectionImpl: svc,
	}
}

func (svc *collectionImpl) Create(ctx context.Context, col *Collection) error {
	query := `INSERT INTO collection (name, workflow_id, run_id, transfer_id, aip_id, original_id, pipeline_id, decision_token, status) VALUES ((?), (?), (?), (?), (?), (?), (?), (?), (?))`
	args := []any{
		col.Name,
		col.WorkflowID,
		col.RunID,
		col.TransferID,
		col.AIPID,
		col.OriginalID,
		col.PipelineID,
		col.DecisionToken,
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

	publishEvent(ctx, svc.events, EventTypeCollectionCreated, col.ID)

	return nil
}

func (svc *collectionImpl) CheckDuplicate(ctx context.Context, id uint) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM collection c1 WHERE c1.name = (SELECT name FROM collection WHERE id = ?) AND c1.id <> ? AND c1.status NOT IN (3, 6))`
	var exists bool
	err := svc.db.GetContext(ctx, &exists, query, id, id)
	if err != nil {
		return false, fmt.Errorf("sql error: %w", err)
	}
	return exists, nil
}

func publishEvent(ctx context.Context, events EventService, eventType string, id uint) {
	// TODO: publish updated collection?
	var item *goacollection.EnduroStoredCollection

	events.PublishEvent(&goacollection.EnduroMonitorUpdate{
		ID:   id,
		Type: eventType,
		Item: item,
	})
}

func (svc *collectionImpl) UpdateWorkflowStatus(ctx context.Context, ID uint, name, workflowID, runID, transferID, aipID, pipelineID string, status Status, storedAt time.Time) error {
	// Ensure that storedAt is reset during retries.
	completedAt := &storedAt
	if status == StatusInProgress {
		completedAt = nil
	}
	if completedAt != nil && completedAt.IsZero() {
		completedAt = nil
	}

	query := `UPDATE collection SET name = (?), workflow_id = (?), run_id = (?), transfer_id = (?), aip_id = (?), pipeline_id = (?), status = (?), completed_at = (?) WHERE id = (?)`
	args := []any{
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

	if _, err := svc.updateRow(ctx, query, args); err != nil {
		return err
	}

	publishEvent(ctx, svc.events, EventTypeCollectionUpdated, ID)

	return nil
}

func (svc *collectionImpl) SetStatus(ctx context.Context, ID uint, status Status) error {
	query := `UPDATE collection SET status = (?) WHERE id = (?)`
	args := []any{
		status,
		ID,
	}

	if _, err := svc.updateRow(ctx, query, args); err != nil {
		return err
	}

	publishEvent(ctx, svc.events, EventTypeCollectionUpdated, ID)

	return nil
}

func (svc *collectionImpl) SetStatusInProgress(ctx context.Context, ID uint, startedAt time.Time) error {
	var query string
	args := []any{StatusInProgress}

	if !startedAt.IsZero() {
		query = `UPDATE collection SET status = (?), started_at = (?) WHERE id = (?)`
		args = append(args, startedAt, ID)
	} else {
		query = `UPDATE collection SET status = (?) WHERE id = (?)`
		args = append(args, ID)
	}

	if _, err := svc.updateRow(ctx, query, args); err != nil {
		return err
	}

	publishEvent(ctx, svc.events, EventTypeCollectionUpdated, ID)

	return nil
}

func (svc *collectionImpl) SetStatusPending(ctx context.Context, ID uint, taskToken []byte) error {
	query := `UPDATE collection SET status = (?), decision_token = (?) WHERE id = (?)`
	args := []any{
		StatusPending,
		taskToken,
		ID,
	}

	if _, err := svc.updateRow(ctx, query, args); err != nil {
		return err
	}

	publishEvent(ctx, svc.events, EventTypeCollectionUpdated, ID)

	return nil
}

func (svc *collectionImpl) SetOriginalID(ctx context.Context, ID uint, originalID string) error {
	query := `UPDATE collection SET original_id = (?) WHERE id = (?)`
	args := []any{
		originalID,
		ID,
	}

	if _, err := svc.updateRow(ctx, query, args); err != nil {
		return err
	}

	publishEvent(ctx, svc.events, EventTypeCollectionUpdated, ID)

	return nil
}

func (svc *collectionImpl) updateRow(ctx context.Context, query string, args []any) (int64, error) {
	query = svc.db.Rebind(query)
	res, err := svc.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("error updating collection: %v", err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("error retrieving rows affected: %v", err)
	}

	return n, nil
}

func (svc *collectionImpl) read(ctx context.Context, ID uint) (*Collection, error) {
	query := "SELECT id, name, workflow_id, run_id, transfer_id, aip_id, original_id, pipeline_id, decision_token, status, CONVERT_TZ(created_at, @@session.time_zone, '+00:00') AS created_at, CONVERT_TZ(started_at, @@session.time_zone, '+00:00') AS started_at, CONVERT_TZ(completed_at, @@session.time_zone, '+00:00') AS completed_at FROM collection WHERE id = (?)"
	args := []any{ID}
	c := Collection{}

	query = svc.db.Rebind(query)
	if err := svc.db.GetContext(ctx, &c, query, args...); err != nil {
		return nil, err
	}

	return &c, nil
}
