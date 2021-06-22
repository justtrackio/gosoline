package log

type Option func(logger *gosoLogger) error

func WithContextFieldsResolver(resolvers ...ContextFieldsResolver) Option {
	return func(logger *gosoLogger) error {
		logger.ctxResolvers = append(logger.ctxResolvers, resolvers...)

		return nil
	}
}

func WithFields(tags map[string]interface{}) Option {
	return func(logger *gosoLogger) error {
		for k, v := range tags {
			logger.data.Fields[k] = v
		}

		return nil
	}
}

func WithHandlers(handler ...Handler) Option {
	return func(logger *gosoLogger) error {
		logger.handlers = append(logger.handlers, handler...)
		return nil
	}
}
