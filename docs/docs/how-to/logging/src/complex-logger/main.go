package main

import (
	"context"
	"fmt"
	"os"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	// 1
	"github.com/justtrackio/gosoline/pkg/log"
)

func main() {
	ctx := context.Background()

	// 2
	handler := log.NewHandlerIoWriter(cfg.New(), log.PriorityDebug, log.FormatterConsole, "main", "15:04:05.000", os.Stdout)
	logger := log.NewLoggerWithInterfaces(clock.NewRealClock(), []log.Handler{handler})

	if err := logger.Option(log.WithContextFieldsResolver(log.ContextFieldsResolver)); err != nil {
		panic(err)
	}

	// 3
	logger.Info(ctx, "log a number %d", 4)

	// 4
	logger.WithChannel("strings").Warn(ctx, "a dangerous string appeared: %s", "foobar")

	// 5
	loggerWithFields := logger.WithFields(log.Fields{
		"b": true,
	})
	loggerWithFields.Debug(ctx, "just some debug line")
	loggerWithFields.Error(ctx, "it happens: %w", fmt.Errorf("should not happen"))

	// 6
	ctx = log.AppendContextFields(ctx, map[string]any{
		"id": 1337,
	})
	logger.Info(ctx, "some info")
}
