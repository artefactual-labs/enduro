package activities

import (
	"fmt"
	"testing"

	wferrors "github.com/artefactual-labs/enduro/internal/workflow/errors"
	"go.uber.org/cadence"
	"gotest.tools/v3/assert"
)

// activityError describes an error assertion, where its zero value is a
// non-error. Its Assert method is used to assert against a given error value.
type activityError struct {
	Message string // Text message.
	NRE     bool   // Process as a NRE error.
}

func (ae activityError) IsZero() bool {
	return ae == activityError{}
}

func (ae activityError) Assert(t *testing.T, err error) {
	t.Helper()

	// Test for a nil error.
	if ae.IsZero() {
		ae.AssertNilError(t, err)
		return
	}

	// Test for a non-retryable error.
	if ae.NRE {
		ae.assertNonRetryableError(t, err)
		return
	}

	// Test for a regular error.
	assert.Error(t, err, ae.Message)
}

func (ae activityError) AssertNilError(t *testing.T, err error) {
	t.Helper()

	details := func(err error) string {
		if err == nil {
			return ""
		}

		perr, ok := err.(*cadence.CustomError)
		if !ok {
			return err.Error()
		}

		var result string
		if err := perr.Details(&result); err != nil {
			return fmt.Sprintf("cannot extract details: %v", err)
		}

		return result
	}

	assert.NilError(t, err, fmt.Sprintf("error is not nil: %s", details(err)))
}

func (ae activityError) assertNonRetryableError(t *testing.T, err error) {
	t.Helper()

	assert.ErrorType(t, err, &cadence.CustomError{})
	assert.Error(t, err, wferrors.NRE)

	// cadence.CustomError has a details field where our test message goes.
	var result string
	perr := err.(*cadence.CustomError)
	assert.NilError(t, perr.Details(&result))
	assert.Equal(t, result, ae.Message)
}
