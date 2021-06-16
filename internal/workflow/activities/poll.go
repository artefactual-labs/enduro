package activities

import (
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/jonboulle/clockwork"
)

var (
	// Our default back-off strategy when polling.
	backoffStrategy backoff.BackOff = backoff.NewConstantBackOff(time.Second * 5)

	// Default deadline when retrying on errors, e.g. HTTP 5xx. Users can
	// override with retryDeadline, a pipeline configuration attribute.
	defaultMaxElapsedTime = time.Minute * 10

	// System clock.
	clock clockwork.Clock = clockwork.NewRealClock()
)
