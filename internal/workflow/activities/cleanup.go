package activities

import (
	"context"
	"fmt"
	"os"
)

// CleanUpActivity removes the contents that we've created in the TS location.
type CleanUpActivity struct{}

func NewCleanUpActivity() *CleanUpActivity {
	return &CleanUpActivity{}
}

type CleanUpActivityParams struct {
	Paths []string
}

func (a *CleanUpActivity) Execute(ctx context.Context, params *CleanUpActivityParams) error {
	if params == nil {
		return fmt.Errorf("error processing parameters: missing")
	}

	for _, p := range params.Paths {
		if err := os.RemoveAll(p); err != nil {
			return fmt.Errorf("error removing path: %v", err)
		}
	}

	return nil
}
