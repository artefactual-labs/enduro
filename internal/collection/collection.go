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
)

type Service interface {
	// Goa returns an implementation of the goacollection Service.
	Goa() goacollection.Service
	Create(context.Context, *Collection) error
	UpdateWorkflowStatus(ctx context.Context, ID uint, name string, workflowID, runID, aipID string, status Status, storedAt time.Time) error
	SetStatus(ctx context.Context, ID uint, status Status) error
	SetStatusInProgress(ctx context.Context, ID uint, startedAt time.Time) error
	SetStatusPending(ctx context.Context, ID uint) error
	CreatePreservationAction(ctx context.Context, pa *PreservationAction) error
}

type collectionImpl struct {
	logger logr.Logger
	db     *sqlx.DB
	tc     temporalsdk_client.Client

	// Destination for events to be published.
	events EventService
}

var _ Service = (*collectionImpl)(nil)

func NewService(logger logr.Logger, db *sql.DB, tc temporalsdk_client.Client) *collectionImpl {
	return &collectionImpl{
		logger: logger,
		db:     sqlx.NewDb(db, "mysql"),
		tc:     tc,
		events: NewEventService(),
	}
}

func (svc *collectionImpl) Goa() goacollection.Service {
	return &goaWrapper{
		collectionImpl: svc,
	}
}

func (svc *collectionImpl) Create(ctx context.Context, col *Collection) error {
	query := `INSERT INTO collection (name, workflow_id, run_id, aip_id, status) VALUES ((?), (?), (?), (?), (?))`
	args := []interface{}{
		col.Name,
		col.WorkflowID,
		col.RunID,
		col.AIPID,
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

func publishEvent(ctx context.Context, events EventService, eventType string, id uint) {
	// TODO: publish updated collection?
	var item *goacollection.EnduroStoredCollection

	events.PublishEvent(&goacollection.EnduroMonitorUpdate{
		ID:   id,
		Type: eventType,
		Item: item,
	})
}

func (svc *collectionImpl) UpdateWorkflowStatus(ctx context.Context, ID uint, name string, workflowID, runID, aipID string, status Status, storedAt time.Time) error {
	// Ensure that storedAt is reset during retries.
	completedAt := &storedAt
	if status == StatusInProgress {
		completedAt = nil
	}
	if completedAt != nil && completedAt.IsZero() {
		completedAt = nil
	}

	query := `UPDATE collection SET name = (?), workflow_id = (?), run_id = (?), aip_id = (?), status = (?), completed_at = (?) WHERE id = (?)`
	args := []interface{}{
		name,
		workflowID,
		runID,
		aipID,
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
	args := []interface{}{
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
	args := []interface{}{StatusInProgress}

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

func (svc *collectionImpl) SetStatusPending(ctx context.Context, ID uint) error {
	query := `UPDATE collection SET status = (?), WHERE id = (?)`
	args := []interface{}{
		StatusPending,
		ID,
	}

	if _, err := svc.updateRow(ctx, query, args); err != nil {
		return err
	}

	publishEvent(ctx, svc.events, EventTypeCollectionUpdated, ID)

	return nil
}

func (svc *collectionImpl) updateRow(ctx context.Context, query string, args []interface{}) (int64, error) {
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
	query := "SELECT id, name, workflow_id, run_id, aip_id, status, CONVERT_TZ(created_at, @@session.time_zone, '+00:00') AS created_at, CONVERT_TZ(started_at, @@session.time_zone, '+00:00') AS started_at, CONVERT_TZ(completed_at, @@session.time_zone, '+00:00') AS completed_at FROM collection WHERE id = (?)"
	args := []interface{}{ID}
	c := Collection{}

	query = svc.db.Rebind(query)
	if err := svc.db.GetContext(ctx, &c, query, args...); err != nil {
		return nil, err
	}

	return &c, nil
}
