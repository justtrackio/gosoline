package application

import (
	"github.com/applike/gosoline/pkg/mon"
	"os"
)

type ErrorHandler func(msg string, args ...interface{})

func WithDefaultErrorHandler(handler ErrorHandler) {
	defaultErrorHandler = handler
}

var defaultErrorHandler = func(msg string, args ...interface{}) {
	logger := mon.NewLogger()
	options := []mon.LoggerOption{
		mon.WithFormat(mon.FormatJson),
		mon.WithTimestampFormat("2006-01-02T15:04:05.999Z07:00"),
		mon.WithHook(mon.NewMetricHook()),
	}

	if err := logger.Option(options...); err != nil {
		logger.Error("can not create logger for default error handler: %w", err)
		os.Exit(1)
	}

	logger.Error(msg, args...)
	os.Exit(1)
}
