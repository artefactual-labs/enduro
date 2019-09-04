package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"time"

	"github.com/artefactual-labs/enduro/internal/collection"
)

// Similar to time.RFC3339 with dashes and colons removed.
const rfc3339forFilename = "20060102.150405.999999999"

type UpdateProductionSystemActivity struct {
	manager *Manager
}

func NewUpdateProductionSystemActivity(m *Manager) *UpdateProductionSystemActivity {
	return &UpdateProductionSystemActivity{manager: m}
}

func (a *UpdateProductionSystemActivity) Execute(ctx context.Context, tinfo *TransferInfo) error {
	// We expect tinfo.StoredAt to have the zero value when the ingestion
	// has failed. Here we set a new value as it is a required field.
	if tinfo.StoredAt.IsZero() {
		tinfo.StoredAt = time.Now().UTC()
	}

	receiptPath, err := hookAttrString(a.manager.Hooks, "prod", "receiptPath")
	if err != nil {
		return fmt.Errorf("error looking up receiptPath configuration attribute: %w", err)
	}

	var filename = fmt.Sprintf("Receipt_%s_%s.json", tinfo.OriginalID, tinfo.StoredAt.Format(rfc3339forFilename))
	receiptPath = path.Join(receiptPath, filename)

	file, err := os.OpenFile(receiptPath, os.O_WRONLY|os.O_CREATE, os.FileMode(0o644))
	if err != nil {
		return nonRetryableError(fmt.Errorf("Error creating receipt file %s: %w", receiptPath, err))
	}

	if err := a.generateReceipt(tinfo, file); err != nil {
		return nonRetryableError(fmt.Errorf("Error writing receipt file %s: %w", receiptPath, err))
	}

	return nil
}

func (a UpdateProductionSystemActivity) generateReceipt(tinfo *TransferInfo, writer io.Writer) error {
	var accepted bool
	var message string

	if tinfo.Status == collection.StatusDone {
		accepted = true
		message = fmt.Sprintf("Package was processed by %s Archivematica pipeline", tinfo.Event.PipelineName)
	} else {
		accepted = false
		message = fmt.Sprintf("Package was not processed successfully")
	}

	receipt := prodSystemReceipt{
		Identifier: tinfo.OriginalID,
		Type:       strings.ToLower(tinfo.Kind),
		Accepted:   accepted,
		Message:    message,
		Timestamp:  tinfo.StoredAt,
	}

	enc := json.NewEncoder(writer)
	enc.SetIndent("", "  ")

	return enc.Encode(receipt)
}

type prodSystemReceipt struct {
	Identifier string    `json:"identifier"` // Original identifier.
	Type       string    `json:"type"`       // Lowercase. E.g. "dpj", "epj", "other" or "avlxml".
	Accepted   bool      `json:"accepted"`   // Whether we have an error during processing.
	Message    string    `json:"message"`    // E.g. "Package was processed by DPJ Archivematica pipeline" or any other error message.
	Timestamp  time.Time `json:"timestamp"`  // RFC3339, e.g. "2006-01-02T15:04:05Z07:00"
}
