package testutil

import (
	"errors"
	"fmt"
	"runtime"
	"testing"

	temporalsdk_temporal "go.temporal.io/sdk/temporal"
	"gotest.tools/v3/assert"
)

// ActivityError describes an error assertion, where its zero value is a
// non-error. Its Assert method is used to assert against a given error value.
type ActivityError struct {
	Message        string // Text message.
	MessageWindows string // Test message specific to the Windows platform.
	NonRetryable   bool   // Process as a non-retryable error.
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
	if ae.NonRetryable {
		var nr bool
		var applicationError *temporalsdk_temporal.ApplicationError
		if errors.As(err, &applicationError) && applicationError.NonRetryable() {
			nr = true
		}
		if !nr {
			t.Error("returned error was expected to be unretryable")
		}
	}

	// Test for a regular error.
	assert.ErrorContains(t, err, ae.message())
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

	assert.NilError(t, err, fmt.Sprintf("error is not nil: %v", err))
}
