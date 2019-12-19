package activities

import (
	"context"

	"github.com/artefactual-labs/enduro/internal/nha"
)

func ParseNameLocalActivity(ctx context.Context, name string) (nha.NameInfo, error) {
	result, err := nha.ParseName(name)

	return *result, err
}
