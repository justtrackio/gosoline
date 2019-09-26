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

	if err := logger.Option(mon.WithFormat(mon.FormatGelf)); err != nil {
		panic(err)
	}

	logger.Fatalf(err, msg, args...)
}
