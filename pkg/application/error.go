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
		mon.WithHook(mon.NewMetricHook()),
		mon.WithStdoutOutput(mon.FormatJson, []string{mon.Fatal}),
	}

	if err := logger.Option(options...); err != nil {
		panic(err)
	}

	logger.Fatalf(err, msg, args...)
}
