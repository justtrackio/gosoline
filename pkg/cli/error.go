package cli

import (
	"github.com/applike/gosoline/pkg/log"
	"os"
)

type ErrorHandler func(msg string, args ...interface{})

func WithDefaultErrorHandler(handler ErrorHandler) {
	defaultErrorHandler = handler
}

var defaultErrorHandler = func(msg string, args ...interface{}) {
	logger := log.NewCliLogger()

	logger.Error(msg, args...)
	os.Exit(1)
}
