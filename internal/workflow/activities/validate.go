package activities

import (
	"context"

	"github.com/artefactual-labs/enduro/internal/validation"
)

type ValidateTransferActivity struct{}

func NewValidateTransferActivity() *ValidateTransferActivity {
	return &ValidateTransferActivity{}
}

type ValidateTransferActivityParams struct {
	Config validation.Config
	Path   string
}

func (a *ValidateTransferActivity) Execute(ctx context.Context, params *ValidateTransferActivityParams) error {
	return validation.ValidateTransfer(params.Config, params.Path)
}
