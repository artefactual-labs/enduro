package activities

import (
	"context"

	"github.com/artefactual-labs/enduro/internal/validation"
)

type ValidateTransferActivity struct {
	Config validation.Config
}

func NewValidateTransferActivity(config validation.Config) *ValidateTransferActivity {
	return &ValidateTransferActivity{Config: config}
}

type ValidateTransferActivityParams struct {
	Path string
}

func (a *ValidateTransferActivity) Execute(ctx context.Context, params *ValidateTransferActivityParams) error {
	return validation.ValidateTransfer(a.Config, params.Path)
}
