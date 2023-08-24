package activities

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/artefactual-labs/enduro/internal/pipeline"
)

// PopulateMetadataActivity populates the DC identifier in the transfer metadata CSV document.
type PopulateMetadataActivity struct {
	pipelineRegistry *pipeline.Registry
}

func NewPopulateMetadataActivity(pipelineRegistry *pipeline.Registry) *PopulateMetadataActivity {
	return &PopulateMetadataActivity{pipelineRegistry: pipelineRegistry}
}

type PopulateMetadataActivityParams struct {
	Path       string
	Identifier string
}

func (a *PopulateMetadataActivity) Execute(ctx context.Context, params *PopulateMetadataActivityParams) error {
	if params == nil || params.Path == "" || params.Identifier == "" {
		return errors.New("unexpected parameters")
	}

	path := filepath.Join(params.Path, "metadata")
	err := os.MkdirAll(path, 0o755)
	if err != nil {
		return err
	}

	path = filepath.Join(path, "metadata.csv")
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o664)
	if err != nil {
		return fmt.Errorf("it was not possible to open the metadata file: %v", err)
	}

	csvw := csv.NewWriter(f)
	_ = csvw.WriteAll(
		[][]string{
			{"parts", "dc.identifier"},
			{"objects", params.Identifier},
		},
	)

	return csvw.Error()
}
