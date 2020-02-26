// +build !fixtures

package fixtures

import (
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
)

func NewFixtureLoader(config cfg.Config, logger mon.Logger) FixtureLoader {
	return &noopFixtureLoader{}
}

type noopFixtureLoader struct {
}

func (n noopFixtureLoader) Load(fixtureSets []*FixtureSet) error {
	return nil // do nothing
}
