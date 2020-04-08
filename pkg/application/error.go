package application

import (
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/mon/daemon"
)

type ErrorHandler func(err error, msg string, args ...interface{})

func WithDefaultErrorHandler(handler ErrorHandler) {
	defaultErrorHandler = handler
}

var defaultErrorHandler = func(err error, msg string, args ...interface{}) {
	logger := mon.NewLogger()
	options := []mon.LoggerOption{
		mon.WithFormat(mon.FormatJson),
		mon.WithTimestampFormat("2006-01-02T15:04:05.999Z07:00"),
		mon.WithHook(daemon.NewMetricHook()),
	}

	if err := logger.Option(options...); err != nil {
		panic(err)
	}

	logger.Fatalf(err, msg, args...)
}
