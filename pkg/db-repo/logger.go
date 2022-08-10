package db_repo

import (
	"context"
	"time"

	"gorm.io/gorm/logger"
)

type noopLogger struct{}

func (l *noopLogger) LogMode(level logger.LogLevel) logger.Interface { return l }

func (l *noopLogger) Info(ctx context.Context, s string, i ...interface{}) {}

func (l *noopLogger) Warn(ctx context.Context, s string, i ...interface{}) {}

func (l *noopLogger) Error(ctx context.Context, s string, i ...interface{}) {}

func (l *noopLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
}
