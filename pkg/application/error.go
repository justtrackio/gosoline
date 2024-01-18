package application

import (
	"os"

	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/metric"
)

type ErrorHandler func(msg string, args ...interface{})

func withDefaultErrorHandler(handler ErrorHandler) {
	defaultErrorHandler = handler
}

var defaultErrorHandler = func(msg string, args ...interface{}) {
	writerHandler := log.NewHandlerIoWriter(log.LevelInfo, []string{}, log.FormatterJson, "2006-01-02T15:04:05.999Z07:00", os.Stdout)
	metricHandler := metric.NewLoggerHandler()

	logger := log.NewLoggerWithInterfaces(clock.Provider, []log.Handler{
		writerHandler,
		metricHandler,
	})

	logger.Error(msg, args...)
	os.Exit(1)
}

func callDefaultErrorHandler(msg string, args ...interface{}) {
	defaultErrorHandler(msg, args...)
}
