package activities

import (
	"context"
	"fmt"
	"os"

	"github.com/artefactual-labs/enduro/internal/workflow/manager"
)

// CleanUpActivity removes the contents that we've created in the TS location.
type CleanUpActivity struct {
	manager *manager.Manager
}

func NewCleanUpActivity(m *manager.Manager) *CleanUpActivity {
	return &CleanUpActivity{manager: m}
}

type CleanUpActivityParams struct {
	FullPath string
}

func (a *CleanUpActivity) Execute(ctx context.Context, params *CleanUpActivityParams) error {
	if params == nil || params.FullPath == "" {
		return fmt.Errorf("error processing parameters: missing or empty")
	}

	if err := os.RemoveAll(params.FullPath); err != nil {
		return fmt.Errorf("error removing transfer directory: %v", err)
	}

	return nil
}
