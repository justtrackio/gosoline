package fixtures

import (
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
)

type FixtureWriter interface {
	WriteFixtures() error
}

type FixtureWriterFactory func(config cfg.Config, logger mon.Logger, reader FixtureReader) FixtureWriter

var fixtureWriters = map[string]FixtureWriterFactory{
	SettingsOutputWriterMysql: NewMySqlFixtureWriter,
}

