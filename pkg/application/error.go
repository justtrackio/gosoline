package application

import (
	"github.com/applike/gosoline/pkg/mon"
)

type ErrorHandler func(err error, msg string, args ...interface{})

func WithDefaultErrorHandler(handler ErrorHandler) {
	defaultErrorHandler = handler
}

var defaultErrorHandler = func(err error, msg string, args ...interface{}) {
	logger := mon.NewLogger()
	options := []mon.LoggerOption{
		mon.WithFormat(mon.FormatJson),
		mon.WithHook(mon.NewMetricHook()),
	}

	if err := logger.Option(options...); err != nil {
		panic(err)
	}

	logger.Fatalf(err, msg, args...)
}
