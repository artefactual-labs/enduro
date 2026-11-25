package activities

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"os"

	"github.com/artefactual-labs/enduro/internal/pipeline"
)

// PopulateMetadataActivity writes transfer metadata for Archivematica ingest.
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

	root, err := os.OpenRoot(params.Path)
	if err != nil {
		return err
	}
	defer root.Close()

	err = root.MkdirAll("metadata", 0o755)
	if err != nil {
		return err
	}

	f, err := root.OpenFile("metadata/metadata.csv", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o664)
	if err != nil {
		return fmt.Errorf("it was not possible to open the metadata file: %v", err)
	}

	csvw := csv.NewWriter(f)
	err = csvw.WriteAll(
		[][]string{
			{"parts", "dc.identifier"},
			{"objects", params.Identifier},
		},
	)

	return errors.Join(err, f.Close())
}
