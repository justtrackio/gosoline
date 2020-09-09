package mon

import (
	"github.com/applike/gosoline/pkg/clock"
	"os"
	"time"
)

type LoggerOption func(logger *logger) error

func WithContextFieldsResolver(resolver ...ContextFieldsResolver) LoggerOption {
	return func(logger *logger) error {
		logger.ctxResolver = append(logger.ctxResolver, resolver...)

		return nil
	}
}

func WithHook(hook LoggerHook) LoggerOption {
	return func(logger *logger) error {
		logger.hooks = append(logger.hooks, hook)

		return nil
	}
}

func WithHandler(handler Handler) LoggerOption {
	return func(logger *logger) error {
		logger.handlers = append(logger.handlers, handler)

		return nil
	}
}

func WithStdoutOutput(format string, levels []string) LoggerOption {
	return func(logger *logger) error {
		stdoutHandler, err := NewIowriterLoggerHandler(clock.NewRealClock(), FormatConsole, os.Stdout, time.RFC3339, levels)
		if err != nil {
			return err
		}

		logger.handlers = append(logger.handlers, stdoutHandler)

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
			logger.data.Fields[k] = v
			logger.data.Tags[k] = v
		}

		return nil
	}
}
