package log

type Option func(logger *gosoLogger) error

func WithContextFieldsResolver(resolvers ...ContextFieldsResolverFunction) Option {
	return func(logger *gosoLogger) error {
		logger.ctxResolvers = append(logger.ctxResolvers, resolvers...)

		return nil
	}
}

func WithFields(tags map[string]any) Option {
	return func(logger *gosoLogger) error {
		logger.data.Fields = mergeFields(logger.data.Fields, tags)

		return nil
	}
}

func WithHandlers(handler ...Handler) Option {
	return func(logger *gosoLogger) error {
		logger.handlers = append(logger.handlers, handler...)

		return nil
	}
}
