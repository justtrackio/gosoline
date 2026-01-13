package log

// Option is a functional option pattern for configuring a GosoLogger.
type Option func(logger *gosoLogger) error

// WithContextFieldsResolver adds custom context resolvers to the logger.
// These resolvers are used to extract fields from the context during logging.
func WithContextFieldsResolver(resolvers ...ContextFieldsResolverFunction) Option {
	return func(logger *gosoLogger) error {
		logger.ctxResolvers = append(logger.ctxResolvers, resolvers...)

		return nil
	}
}

// WithFields adds a default set of fields to every log entry created by this logger.
func WithFields(tags map[string]any) Option {
	return func(logger *gosoLogger) error {
		logger.data.Fields = mergeFields(logger.data.Fields, tags)

		return nil
	}
}

// WithHandlers adds additional log handlers to the logger.
func WithHandlers(handler ...Handler) Option {
	return func(logger *gosoLogger) error {
		logger.handlers = append(logger.handlers, handler...)

		return nil
	}
}

// WithSamplingEnabled enables or disables log sampling for the logger.
func WithSamplingEnabled(enabled bool) Option {
	return func(logger *gosoLogger) error {
		logger.samplingEnabled = enabled

		return nil
	}
}
