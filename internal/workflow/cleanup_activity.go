package workflow

import (
	"context"
	"fmt"
	"os"
)

type CleanUpActivity struct {
	manager *Manager
}

func NewCleanUpActivity(m *Manager) *CleanUpActivity {
	return &CleanUpActivity{manager: m}
}

type CleanUpActivityParams struct {
	FullPath string
}

func (a *CleanUpActivity) Execute(ctx context.Context, params *CleanUpActivityParams) error {
	if params == nil || params.FullPath == "" {
		return errMissingParameters
	}

	if err := os.RemoveAll(params.FullPath); err != nil {
		return fmt.Errorf("error removing transfer directory: %v", err)
	}

	return nil
}
