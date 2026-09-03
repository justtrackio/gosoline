package metric

import "fmt"

// The dimension keys shared across packages. A key an OpenTelemetry semantic convention defines is
// spelled the way the convention spells it; every other key is authored in the canonical form metric
// names use.
const (
	// DimensionErrorType is the semantic-convention attribute identifying what went wrong. It is set on
	// the metric recording an operation, so a failure does not need a metric of its own.
	DimensionErrorType = "error.type"
	// DimensionMessagingDestination is the semantic-convention attribute naming the destination a
	// message is read from or written to: a queue, a topic, a stream or a list.
	DimensionMessagingDestination = "messaging.destination.name"
	// DimensionModelId identifies the model an operation applies to.
	DimensionModelId = "model.id"
)

// ErrorType returns the value for DimensionErrorType describing an error, which is its type rather
// than its message, because a message carries unbounded cardinality. A nil error has no type.
func ErrorType(err error) string {
	if err == nil {
		return ""
	}

	return fmt.Sprintf("%T", err)
}
