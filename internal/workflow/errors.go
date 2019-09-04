package workflow

import (
	"go.uber.org/cadence"
	"go.uber.org/cadence/.gen/go/shared"
	"go.uber.org/cadence/workflow"
)

func HeartbeatTimeoutError(err error, details interface{}) bool {
	timeoutErr, ok := err.(*workflow.TimeoutError)
	if !ok {
		return false
	}

	if timeoutErr.TimeoutType() != shared.TimeoutTypeHeartbeat {
		return false
	}

	if timeoutErr.HasDetails() {
		_ = timeoutErr.Details(&details)
	}

	return true
}

const NRE = "non retryable error"

func nonRetryableError(err error) error {
	return cadence.NewCustomError(NRE, err.Error())
}
