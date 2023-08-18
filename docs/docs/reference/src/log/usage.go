package main

import (
	"context"
	"fmt"
	"os"

	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/log"
)

func Usage() {
	handler := log.NewHandlerIoWriter(log.LevelDebug, []string{}, log.FormatterConsole, "15:04:05.000", os.Stdout)
	logger := log.NewLoggerWithInterfaces(clock.NewRealClock(), []log.Handler{handler})

	if err := logger.Option(log.WithContextFieldsResolver(log.ContextLoggerFieldsResolver)); err != nil {
		panic(err)
	}

	logger.Info("log a number %d", 4)
	logger.WithChannel("strings").Warn("a dangerous string appeared: %s", "foobar")

	loggerWithFields := logger.WithFields(log.Fields{
		"b": true,
	})
	loggerWithFields.Debug("just some debug line")
	loggerWithFields.Error("it happens: %w", fmt.Errorf("should not happen"))

	ctx := context.Background()
	ctx = log.AppendLoggerContextField(ctx, map[string]interface{}{
		"id": 1337,
	})

	contextAwareLogger := logger.WithContext(ctx)
	contextAwareLogger.Info("some info")
}
