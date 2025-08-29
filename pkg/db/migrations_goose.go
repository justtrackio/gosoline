package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/pressly/goose/v3"
)

func runMigrationGoose(ctx context.Context, logger log.Logger, settings *Settings, db *sql.DB) error {
	var err error

	goose.SetLogger(newGooseLogger(ctx, logger))

	if err = goose.SetDialect(settings.Driver); err != nil {
		return fmt.Errorf("can not set db dialect: %w", err)
	}

	if err = goose.Up(db, settings.Migrations.Path, goose.WithAllowMissing()); err != nil {
		return fmt.Errorf("can not run up migrations from path %s: %w", settings.Migrations.Path, err)
	}

	return nil
}

type gooseLogger struct {
	ctx    context.Context
	logger log.Logger
}

func newGooseLogger(ctx context.Context, logger log.Logger) goose.Logger {
	return gooseLogger{
		ctx:    ctx,
		logger: logger,
	}
}

func (g gooseLogger) Fatal(v ...any) {
	g.logger.Error(g.ctx, fmt.Sprint(v...))
}

func (g gooseLogger) Fatalf(format string, v ...any) {
	g.logger.Error(g.ctx, format, v...)
}

func (g gooseLogger) Print(v ...any) {
	g.logger.Info(g.ctx, fmt.Sprint(v...))
}

func (g gooseLogger) Println(v ...any) {
	g.logger.Info(g.ctx, fmt.Sprint(v...))
}

func (g gooseLogger) Printf(format string, v ...any) {
	g.logger.Info(g.ctx, format, v...)
}
