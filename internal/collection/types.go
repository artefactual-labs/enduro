package collection

import (
	"database/sql"
	"time"

	goacollection "github.com/artefactual-labs/enduro/internal/api/gen/collection"
)

// Collection represents a collection in the collection table.
type Collection struct {
	ID            uint   `db:"id"`
	Name          string `db:"name"`
	WorkflowID    string `db:"workflow_id"`
	RunID         string `db:"run_id"`
	TransferID    string `db:"transfer_id"`
	AIPID         string `db:"aip_id"`
	OriginalID    string `db:"original_id"`
	PipelineID    string `db:"pipeline_id"`
	DecisionToken []byte `db:"decision_token"`
	Status        Status `db:"status"`

	// It defaults to CURRENT_TIMESTAMP(6) so populated as soon as possible.
	CreatedAt time.Time `db:"created_at"`

	// Nullable, populated as soon as processing starts.
	StartedAt sql.NullTime `db:"started_at"`

	// Nullable, populated as soon as ingest completes.
	CompletedAt sql.NullTime `db:"completed_at"`

	// Nullable, populated when Enduro confirms the AIP exists in storage.
	AIPStoredAt sql.NullTime `db:"aip_stored_at"`

	// Nullable, populated when Enduro reconciles storage state.
	ReconciliationStatus sql.NullString `db:"reconciliation_status"`

	// Nullable, populated whenever Enduro checks storage state.
	ReconciliationCheckedAt sql.NullTime `db:"reconciliation_checked_at"`

	// Nullable, populated when reconciliation cannot complete cleanly.
	ReconciliationError sql.NullString `db:"reconciliation_error"`
}

// GoaSummary returns the API representation used by collection lists.
func (c Collection) GoaSummary() *goacollection.EnduroStoredCollection {
	col := goacollection.EnduroStoredCollection{
		ID:          c.ID,
		Name:        formatOptionalString(c.Name),
		WorkflowID:  formatOptionalString(c.WorkflowID),
		RunID:       formatOptionalString(c.RunID),
		TransferID:  formatOptionalString(c.TransferID),
		AipID:       formatOptionalString(c.AIPID),
		OriginalID:  formatOptionalString(c.OriginalID),
		PipelineID:  formatOptionalString(c.PipelineID),
		Status:      c.Status.String(),
		CreatedAt:   formatTime(c.CreatedAt),
		StartedAt:   formatOptionalTime(c.StartedAt),
		CompletedAt: formatOptionalTime(c.CompletedAt),
	}

	return &col
}

// GoaDetail returns the API representation used by the collection detail view.
func (c Collection) GoaDetail() *goacollection.EnduroDetailedStoredCollection {
	col := goacollection.EnduroDetailedStoredCollection{
		ID:                      c.ID,
		Name:                    formatOptionalString(c.Name),
		WorkflowID:              formatOptionalString(c.WorkflowID),
		RunID:                   formatOptionalString(c.RunID),
		TransferID:              formatOptionalString(c.TransferID),
		AipID:                   formatOptionalString(c.AIPID),
		OriginalID:              formatOptionalString(c.OriginalID),
		PipelineID:              formatOptionalString(c.PipelineID),
		Status:                  c.Status.String(),
		CreatedAt:               formatTime(c.CreatedAt),
		StartedAt:               formatOptionalTime(c.StartedAt),
		CompletedAt:             formatOptionalTime(c.CompletedAt),
		AipStoredAt:             formatOptionalTime(c.AIPStoredAt),
		ReconciliationStatus:    formatOptionalNullString(c.ReconciliationStatus),
		ReconciliationCheckedAt: formatOptionalTime(c.ReconciliationCheckedAt),
		ReconciliationError:     formatOptionalNullString(c.ReconciliationError),
	}

	return &col
}

// formatOptionalString returns the nil value when the string is empty.
func formatOptionalString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// formatOptionalTime returns the nil value when the value is NULL in the db.
func formatOptionalTime(nt sql.NullTime) *string {
	var res *string
	if nt.Valid {
		f := formatTime(nt.Time)
		res = &f
	}
	return res
}

// formatOptionalNullString returns the nil value when the value is NULL in the db.
func formatOptionalNullString(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	return &ns.String
}

// formatTime returns an empty string when t has the zero value.
func formatTime(t time.Time) string {
	var ret string
	if !t.IsZero() {
		ret = t.Format(time.RFC3339)
	}
	return ret
}
