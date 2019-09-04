package collection

import (
	"database/sql"
	"time"

	goacollection "github.com/artefactual-labs/enduro/internal/api/gen/collection"
)

// Collection represents a collection in the collection table.
type Collection struct {
	ID         uint   `db:"id"`
	Name       string `db:"name"`
	WorkflowID string `db:"workflow_id"`
	RunID      string `db:"run_id"`
	TransferID string `db:"transfer_id"`
	AIPID      string `db:"aip_id"`
	OriginalID string `db:"original_id"`
	Status     Status `db:"status"`

	// It defaults to CURRENT_TIMESTAMP(6) so populated as soon as possible.
	CreatedAt time.Time `db:"created_at"`

	// Nullable and only populated as soon as ingest completes.
	CompletedAt sql.NullTime `db:"completed_at"`
}

// Goa returns the API representation of the collection.
func (c Collection) Goa() *goacollection.EnduroStoredCollection {
	col := goacollection.EnduroStoredCollection{
		ID:          c.ID,
		Name:        formatOptionalString(c.Name),
		WorkflowID:  formatOptionalString(c.WorkflowID),
		RunID:       formatOptionalString(c.RunID),
		TransferID:  formatOptionalString(c.TransferID),
		AipID:       formatOptionalString(c.AIPID),
		OriginalID:  formatOptionalString(c.OriginalID),
		Status:      c.Status.String(),
		CreatedAt:   formatTime(c.CreatedAt),
		CompletedAt: formatOptionalTime(c.CompletedAt),
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
		var f = formatTime(nt.Time)
		res = &f
	}
	return res
}

// formatTime returns an empty string when t has the zero value.
func formatTime(t time.Time) string {
	var ret string
	if !t.IsZero() {
		ret = t.Format(time.RFC3339)
	}
	return ret
}
