package collection

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	goacollection "github.com/artefactual-labs/enduro/internal/api/gen/collection"
	"github.com/artefactual-labs/enduro/internal/pipeline"
	"github.com/go-logr/logr"
	goahttp "goa.design/goa/v3/http"

	"github.com/jmoiron/sqlx"
	cadenceclient "go.uber.org/cadence/client"
)

//go:generate mockgen  -destination=./fake/mock_collection.go -package=fake github.com/artefactual-labs/enduro/internal/collection Service

type Service interface {
	// Goa returns an implementation of the goacollection Service.
	Goa() goacollection.Service

	Create(context.Context, *Collection) error
	UpdateWorkflowStatus(ctx context.Context, ID uint, name string, workflowID, runID, transferID, aipID, pipelineID string, status Status, storedAt time.Time) error
	SetStatus(ctx context.Context, ID uint, status Status) error
	SetStatusInProgress(ctx context.Context, ID uint, startedAt time.Time) error
	SetOriginalID(ctx context.Context, ID uint, originalID string) error

	// HTTPDownload returns a HTTP handler that serves the package over HTTP.
	//
	// TODO: this service is meant to be agnostic to protocols. But I haven't
	// found a way in goagen to have my service write directly to the HTTP
	// response writer. Ideally, our goacollection.Service would have a new
	// method that takes a io.Writer (e.g. http.ResponseWriter).
	HTTPDownload(mux goahttp.Muxer, dec func(r *http.Request) goahttp.Decoder) http.HandlerFunc
}

type collectionImpl struct {
	logger logr.Logger
	db     *sqlx.DB
	cc     cadenceclient.Client

	registry *pipeline.Registry

	// downloadProxy generates a reverse proxy on each download.
	downloadProxy *downloadReverseProxy
}

var _ Service = (*collectionImpl)(nil)

func NewService(logger logr.Logger, db *sql.DB, cc cadenceclient.Client, registry *pipeline.Registry) *collectionImpl {
	return &collectionImpl{
		logger:        logger,
		db:            sqlx.NewDb(db, "mysql"),
		cc:            cc,
		registry:      registry,
		downloadProxy: newDownloadReverseProxy(logger),
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

	_, err := svc.updateRow(ctx, query, args)

	return err
}

func (svc *collectionImpl) SetStatus(ctx context.Context, ID uint, status Status) error {
	var query = `UPDATE collection SET status = (?) WHERE id = (?)`
	var args = []interface{}{
		status,
		ID,
	}

	_, err := svc.updateRow(ctx, query, args)

	return err
}

func (svc *collectionImpl) SetStatusInProgress(ctx context.Context, ID uint, startedAt time.Time) error {
	var query = `UPDATE collection SET status = (?), started_at = (?) WHERE id = (?)`
	var args = []interface{}{
		StatusInProgress,
		startedAt,
		ID,
	}

	_, err := svc.updateRow(ctx, query, args)

	return err
}

func (svc *collectionImpl) SetOriginalID(ctx context.Context, ID uint, originalID string) error {
	var query = `UPDATE collection SET original_id = (?) WHERE id = (?)`
	var args = []interface{}{
		originalID,
		ID,
	}

	_, err := svc.updateRow(ctx, query, args)

	return err
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
