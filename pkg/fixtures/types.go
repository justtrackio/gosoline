package fixtures

import (
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/log"
)

type FixtureSet struct {
	Enabled  bool
	Purge    bool
	Writer   FixtureWriterFactory
	Fixtures []interface{}
}

type FixtureLoader interface {
	Load(fixtureSets []*FixtureSet) error
}

type FixtureWriter interface {
	Purge() error
	Write(fixture *FixtureSet) error
}

type FixtureWriterFactory func(config cfg.Config, logger log.Logger) (FixtureWriter, error)
