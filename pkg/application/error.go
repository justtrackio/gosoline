package application

import (
	"os"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/metric"
)

type ErrorHandler func(msg string, args ...any)

func withDefaultErrorHandler(handler ErrorHandler) {
	defaultErrorHandler = handler
}

var defaultErrorHandler = func(msg string, args ...any) {
	writerHandler := log.NewHandlerIoWriter(cfg.New(), log.LevelInfo, log.FormatterJson, "app", "2006-01-02T15:04:05.999Z07:00", os.Stdout)
	metricHandler := metric.NewLoggerHandler()

	logger := log.NewLoggerWithInterfaces(clock.Provider, []log.Handler{
		writerHandler,
		metricHandler,
	})

	logger.Error(msg, args...)
	os.Exit(1)
}
