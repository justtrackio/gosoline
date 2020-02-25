package fixtures

import (
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
)

type FixtureWriter interface {
	WriteFixtures(fixture *FixtureSet) error
}

type FixtureWriterFactory func(config cfg.Config, logger mon.Logger) FixtureWriter
