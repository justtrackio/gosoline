package application

import (
	"github.com/applike/gosoline/pkg/clock"
	"github.com/applike/gosoline/pkg/log"
	"github.com/applike/gosoline/pkg/metric"
	"os"
)

type ErrorHandler func(msg string, args ...interface{})

func WithDefaultErrorHandler(handler ErrorHandler) {
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
