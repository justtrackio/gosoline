package mon

import (
	"fmt"
	"io"
)

type LoggerOption func(logger *logger) error

func WithContextFieldsResolver(resolver ...ContextFieldsResolver) LoggerOption {
	return func(logger *logger) error {
		logger.ctxResolver = append(logger.ctxResolver, resolver...)
		return nil
	}
}

func WithFormat(format string) LoggerOption {
	return func(logger *logger) error {
		if _, ok := formatters[format]; !ok {
			return fmt.Errorf("unknown logger format: %s", format)
		}

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

func WithOutput(output io.Writer) LoggerOption {
	return func(logger *logger) error {
		logger.output = output
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
