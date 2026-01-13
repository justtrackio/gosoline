package main

import (
	"context"
	"os"

	// 1
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

func main() {
	ctx := context.Background()
	// 2
	logHandler := log.NewHandlerIoWriter(
		cfg.New(), log.PriorityInfo, log.FormatterConsole, "main", "15:04:05.000", os.Stdout,
	)

	// 3
	loggerOptions := []log.Option{log.WithHandlers(logHandler)}

	// 4
	logger := log.NewLogger()

	// 5
	if err := logger.Option(loggerOptions...); err != nil {
		logger.Error(ctx, "Failed to apply logger options: %w", err)
		os.Exit(1)
	}

	logger.Info(ctx, "Message")
}
