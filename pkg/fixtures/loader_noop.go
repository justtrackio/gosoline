//go:build !fixtures
// +build !fixtures

package fixtures

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

type noopFixtureLoader struct {
	logger log.Logger
}

func NewFixtureLoader(ctx context.Context, config cfg.Config, logger log.Logger) FixtureLoader {
	return &noopFixtureLoader{
		logger: logger.WithChannel("fixture_loader"),
	}
}

func (n *noopFixtureLoader) Load(ctx context.Context, fixtureSets []FixtureSet) error {
	n.logger.Info("fixtureSets loading disabled, to enable it use the 'fixtures' build tag")

	return nil
}
