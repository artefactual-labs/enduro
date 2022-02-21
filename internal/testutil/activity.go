package testutil

import (
	"fmt"
	"runtime"
	"testing"

	cadencesdk "go.uber.org/cadence"
	"gotest.tools/v3/assert"

	wferrors "github.com/artefactual-labs/enduro/internal/workflow/errors"
)

// ActivityError describes an error assertion, where its zero value is a
// non-error. Its Assert method is used to assert against a given error value.
type ActivityError struct {
	Message        string // Text message.
	MessageWindows string // Test message specific to the Windows platform.
	NRE            bool   // Process as a NRE error.
}

func (ae ActivityError) IsZero() bool {
	return ae == ActivityError{}
}

func (ae ActivityError) Assert(t *testing.T, err error) {
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
	assert.Error(t, err, ae.message())
}

func (ae ActivityError) message() string {
	message := ae.Message

	if runtime.GOOS == "windows" && ae.MessageWindows != "" {
		message = ae.MessageWindows
	}

	return message
}

func (ae ActivityError) AssertNilError(t *testing.T, err error) {
	t.Helper()

	details := func(err error) string {
		if err == nil {
			return ""
		}

		perr, ok := err.(*cadencesdk.CustomError)
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

func (ae ActivityError) assertNonRetryableError(t *testing.T, err error) {
	t.Helper()

	assert.ErrorType(t, err, &cadencesdk.CustomError{})
	assert.Error(t, err, wferrors.NRE)

	// cadence.CustomError has a details field where our test message goes.
	var result string
	perr := err.(*cadencesdk.CustomError)
	assert.NilError(t, perr.Details(&result))

	if ae.message() != "" {
		assert.Equal(t, result, ae.message())
	}
}
