//go:build !fixtures
// +build !fixtures

package fixtures

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/fixtures/writers"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

type noopFixtureLoader struct {
	logger log.Logger
}

func NewFixtureLoader(ctx context.Context, config cfg.Config, logger log.Logger) writers.FixtureLoader {
	return &noopFixtureLoader{
		logger: logger.WithChannel("fixture_loader"),
	}
}

func (n noopFixtureLoader) Load(ctx context.Context, fixtureSets []*writers.FixtureSet) error {
	n.logger.Info("fixtures loading disabled, to enable it use the 'fixtures' build tag")
	return nil
}
