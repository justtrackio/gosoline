package lambda

import (
	"context"
	"os"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/metric"
)

type ErrorHandler func(ctx context.Context, msg string, args ...any)

func WithDefaultErrorHandler(handler ErrorHandler) {
	defaultErrorHandler = handler
}

var defaultErrorHandler = func(ctx context.Context, msg string, args ...any) {
	writerHandler := log.NewHandlerIoWriter(cfg.New(), log.PriorityInfo, log.FormatterJson, "lambda", "2006-01-02T15:04:05.999Z07:00", os.Stdout)
	metricHandler := metric.NewLoggerHandler()

	logger := log.NewLoggerWithInterfaces(clock.Provider, []log.Handler{
		writerHandler,
		metricHandler,
	})

	logger.Error(ctx, msg, args...)
	os.Exit(1)
}
