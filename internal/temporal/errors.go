package temporal

import (
	"errors"

	temporalsdk_temporal "go.temporal.io/sdk/temporal"
)

func NewNonRetryableError(err error) error {
	return temporalsdk_temporal.NewNonRetryableApplicationError(err.Error(), "", nil, nil)
}

func NonRetryableError(err error) bool {
	applicationError := &temporalsdk_temporal.ApplicationError{}
	if !errors.As(err, &applicationError) {
		return false
	}
	return applicationError.NonRetryable()
}
