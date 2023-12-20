package main

import (
	"context"
	"fmt"
	"os"

	"github.com/justtrackio/gosoline/pkg/clock"
	// 1
	"github.com/justtrackio/gosoline/pkg/log"
)

func main() {
	ctx := context.Background()

	// 2
	handler := log.NewHandlerIoWriter(log.LevelDebug, []string{}, log.FormatterConsole, "15:04:05.000", os.Stdout)
	logger := log.NewLoggerWithInterfaces(clock.NewRealClock(), []log.Handler{handler})

	if err := logger.Option(log.WithContextFieldsResolver(log.ContextFieldsResolver)); err != nil {
		panic(err)
	}

	// 3
	logger.Info("log a number %d", 4)

	// 4
	logger.WithChannel("strings").Warn("a dangerous string appeared: %s", "foobar")

	// 5
	loggerWithFields := logger.WithFields(log.Fields{
		"b": true,
	})
	loggerWithFields.Debug("just some debug line")
	loggerWithFields.Error("it happens: %w", fmt.Errorf("should not happen"))

	// 6
	ctx = log.AppendContextFields(ctx, map[string]interface{}{
		"id": 1337,
	})
	contextAwareLogger := logger.WithContext(ctx)
	contextAwareLogger.Info("some info")
}
