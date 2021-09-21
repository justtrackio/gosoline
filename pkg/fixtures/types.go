package fixtures

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

type FixtureSet struct {
	Enabled  bool
	Purge    bool
	Writer   FixtureWriterFactory
	Fixtures []interface{}
}

type FixtureLoader interface {
	Load(ctx context.Context, fixtureSets []*FixtureSet) error
}

type FixtureWriter interface {
	Purge(ctx context.Context) error
	Write(ctx context.Context, fixture *FixtureSet) error
}

type FixtureWriterFactory func(ctx context.Context, config cfg.Config, logger log.Logger) (FixtureWriter, error)
