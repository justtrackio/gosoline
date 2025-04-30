package health

import "time"

type HealthCheckSettings struct {
	// Timeout describes the amount of time until an input is considered stuck and unhealthy. An input has to make at least some kind of progress
	// before the timeout is reached. The following actions are considered progress:
	//
	// - Fetching a batch of messages from its source. An empty batch counts as progress
	// - Feeding a single message to a downstream consumer. An input doesn't turn unhealthy just because the consumer takes too long to process
	//   the batch as long as a single message is processed before the timeout triggers
	// - Trying to acquire resources. If we are waiting for another task to release a resource, we don't turn unhealthy as long as we request the
	//   resource from time to time
	Timeout time.Duration `cfg:"timeout" default:"5m" validate:"min=1"`
}
