package errors

import (
	cadencesdk "go.uber.org/cadence"
	cadencesdk_gen_shared "go.uber.org/cadence/.gen/go/shared"
	cadencesdk_workflow "go.uber.org/cadence/workflow"
)

const NRE = "non retryable error"

func NonRetryableError(err error) error {
	return cadencesdk.NewCustomError(NRE, err.Error())
}

// HeartbeatTimeoutError determines if a given error is a heartbeat error.
func HeartbeatTimeoutError(err error, details interface{}) bool {
	timeoutErr, ok := err.(*cadencesdk_workflow.TimeoutError)
	if !ok {
		return false
	}

	if timeoutErr.TimeoutType() != cadencesdk_gen_shared.TimeoutTypeHeartbeat {
		return false
	}

	if timeoutErr.HasDetails() {
		_ = timeoutErr.Details(&details)
	}

	return true
}
