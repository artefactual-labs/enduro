package activities

import (
	"context"

	"github.com/artefactual-labs/enduro/internal/sdps"
)

type ValidatePackageActivity struct{}

func NewValidatePackageActivity() *ValidatePackageActivity {
	return &ValidatePackageActivity{}
}

func (a *ValidatePackageActivity) Execute(ctx context.Context, path string) error {
	_, err := sdps.OpenBatch(path)
	return err
}
