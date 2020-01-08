package mon

type LoggerOption func(logger *logger) error

func WithContextFieldsResolver(resolver ...ContextFieldsResolver) LoggerOption {
	return func(logger *logger) error {
		logger.ctxResolver = append(logger.ctxResolver, resolver...)
		return nil
	}
}

func WithFormat(format string) LoggerOption {
	return func(logger *logger) error {
		logger.format = format
		return nil
	}
}

func WithHook(hook LoggerHook) LoggerOption {
	return func(logger *logger) error {
		logger.hooks = append(logger.hooks, hook)
		return nil
	}
}

func WithLevel(level string) LoggerOption {
	return func(logger *logger) error {
		logger.level = levelPriority(level)
		return nil
	}
}

func WithTags(tags map[string]interface{}) LoggerOption {
	return func(logger *logger) error {
		for k, v := range tags {
			logger.data.fields[k] = v
			logger.data.tags[k] = v
		}

		return nil
	}
}

func WithTimestampFormat(format string) LoggerOption {
	return func(logger *logger) error {
		logger.timestampFormat = format
		return nil
	}
}
