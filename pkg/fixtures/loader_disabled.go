package fixtures

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/log"
)

type disabledFixtureLoader struct {
	logger log.Logger
}

func NewFixtureLoaderDisabled(logger log.Logger) FixtureLoader {
	return &disabledFixtureLoader{
		logger: logger.WithChannel("fixture_loader"),
	}
}

func (l *disabledFixtureLoader) Load(ctx context.Context) error {
	l.logger.Info(ctx, "fixture loader is not enabled")

	return nil
}
