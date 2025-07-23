package main

import (
	"os"

	// 1
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

func main() {
	// 2
	logHandler := log.NewHandlerIoWriter(
		cfg.New(), log.LevelInfo, log.FormatterConsole, "main", "", os.Stdout,
	)

	// 3
	loggerOptions := []log.Option{log.WithHandlers(logHandler)}

	// 4
	logger := log.NewLogger()

	// 5
	if err := logger.Option(loggerOptions...); err != nil {
		logger.Error("Failed to apply logger options: %w", err)
		os.Exit(1)
	}

	logger.Info("Message")
}
