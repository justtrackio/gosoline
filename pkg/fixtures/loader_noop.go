// +build !fixtures

package fixtures

import (
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
)

type noopFixtureLoader struct {
	logger mon.Logger
}

func NewFixtureLoader(config cfg.Config, logger mon.Logger) FixtureLoader {
	return &noopFixtureLoader{
		logger: logger.WithChannel("fixture_loader"),
	}
}

func (n noopFixtureLoader) Load(fixtureSets []*FixtureSet) error {
	n.logger.Info("fixtures loading disabled, to enable it use the 'fixtures' build tag")
	return nil
}
