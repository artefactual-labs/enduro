package activities

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"time"

	"github.com/artefactual-labs/enduro/internal/collection"
	wferrors "github.com/artefactual-labs/enduro/internal/workflow/errors"
	"github.com/artefactual-labs/enduro/internal/workflow/manager"
)

// Similar to time.RFC3339 with dashes and colons removed.
const rfc3339forFilename = "20060102.150405.999999999"

type UpdateProductionSystemActivity struct {
	manager *manager.Manager
}

func NewUpdateProductionSystemActivity(m *manager.Manager) *UpdateProductionSystemActivity {
	return &UpdateProductionSystemActivity{manager: m}
}

type UpdateProductionSystemActivityParams struct {
	OriginalID   string
	Kind         string
	StoredAt     time.Time
	PipelineName string
	Status       collection.Status
}

func (a *UpdateProductionSystemActivity) Execute(ctx context.Context, params *UpdateProductionSystemActivityParams) error {
	if params.OriginalID == "" {
		return wferrors.NonRetryableError(errors.New("OriginalID is missing or empty"))
	}

	var err error
	params.Kind, err = convertKind(params.Kind)
	if err != nil {
		return wferrors.NonRetryableError(fmt.Errorf("error validating kind attribute: %v", err))
	}

	// We expect tinfo.StoredAt to have the zero value when the ingestion
	// has failed. Here we set a new value as it is a required field.
	if params.StoredAt.IsZero() {
		params.StoredAt = time.Now().UTC()
	}

	receiptPath, err := manager.HookAttrString(a.manager.Hooks, "prod", "receiptPath")
	if err != nil {
		return wferrors.NonRetryableError(fmt.Errorf("error looking up receiptPath configuration attribute: %v", err))
	}

	var filename = fmt.Sprintf("Receipt_%s_%s.json", params.OriginalID, params.StoredAt.Format(rfc3339forFilename))
	receiptPath = path.Join(receiptPath, filename)

	file, err := os.OpenFile(receiptPath, os.O_WRONLY|os.O_CREATE, os.FileMode(0o644))
	if err != nil {
		return wferrors.NonRetryableError(fmt.Errorf("error creating receipt file: %v", err))
	}

	if err := a.generateReceipt(params, file); err != nil {
		return wferrors.NonRetryableError(fmt.Errorf("error writing receipt file %s: %v", receiptPath, err))
	}

	return nil
}

func (a UpdateProductionSystemActivity) generateReceipt(params *UpdateProductionSystemActivityParams, writer io.Writer) error {
	var accepted bool
	var message string

	if params.Status == collection.StatusDone {
		accepted = true
		message = fmt.Sprintf("Package was processed by Archivematica pipeline %s", params.PipelineName)
	} else {
		accepted = false
		message = fmt.Sprintf("Package was not processed successfully")
	}

	receipt := prodSystemReceipt{
		Identifier: params.OriginalID,
		Type:       strings.ToLower(params.Kind),
		Accepted:   accepted,
		Message:    message,
		Timestamp:  params.StoredAt,
	}

	enc := json.NewEncoder(writer)
	enc.SetIndent("", "  ")

	return enc.Encode(receipt)
}

type prodSystemReceipt struct {
	Identifier string    `json:"identifier"` // Original identifier.
	Type       string    `json:"type"`       // Lowercase. E.g. "dpj", "epj", "other" or "avlxml".
	Accepted   bool      `json:"accepted"`   // Whether we have an error during processing.
	Message    string    `json:"message"`    // E.g. "Package was processed by Archivematica pipeline am" or any other error message.
	Timestamp  time.Time `json:"timestamp"`  // RFC3339, e.g. "2006-01-02T15:04:05Z07:00"
}
