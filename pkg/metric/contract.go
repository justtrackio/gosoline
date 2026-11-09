package metric

// The dimension keys shared across packages. Every key is authored in the canonical form metric names
// use and carries no namespace prefix, because the metric it is attached to already names its
// subsystem. DimensionErrorType is the one key gosoline takes verbatim from an OpenTelemetry semantic
// convention, because its values are what makes an operation metric self-describing.
const (
	// DimensionErrorType is the semantic-convention attribute identifying what went wrong. It is set on
	// the metric recording an operation, so a failure does not need a metric of its own. Its value names
	// the action that failed (for example send_failed, connect_failed), never the Go error type, which
	// would carry unbounded cardinality.
	DimensionErrorType = "error.type"
	// DimensionModelId identifies the model an operation applies to.
	DimensionModelId = "model.id"
)
