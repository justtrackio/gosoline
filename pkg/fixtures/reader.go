package fixtures

import (
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
)

type FixtureReader interface {
	ReadFixtures() (interface{}, error)
}

type FixtureReaderFactory func(config cfg.Config, logger mon.Logger) FixtureReader

var fixtureReaders = map[string]FixtureReaderFactory{
	SettingsInputReaderJson: NewJsonFixtureReader,
}