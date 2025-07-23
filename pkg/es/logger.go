package es

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/log"
)

type gosoLogger struct {
	logger log.Logger
}

func NewLogger(logger log.Logger) Logger {
	return gosoLogger{
		logger: logger,
	}
}

func (g gosoLogger) Debug(format string, args ...any) {
	g.logger.Debug(context.Background(), format, args...)
}

func (g gosoLogger) Info(format string, args ...any) {
	g.logger.Info(context.Background(), format, args...)
}
