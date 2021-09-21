package cli

import (
	"os"

	"github.com/justtrackio/gosoline/pkg/log"
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
