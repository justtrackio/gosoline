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

type FixtureBuilderFactory func(ctx context.Context) (FixtureBuilder, error)

type FixtureBuilder interface {
	Fixtures() []*FixtureSet
}

type FixtureLoader interface {
	Load(ctx context.Context, fixtureSets []*FixtureSet) error
}

type FixtureWriter interface {
	Purge(ctx context.Context) error
	Write(ctx context.Context, fixture *FixtureSet) error
}

type FixtureWriterFactory func(ctx context.Context, config cfg.Config, logger log.Logger) (FixtureWriter, error)

type simpleFixtureBuilder struct {
	fixtureSets []*FixtureSet
}

func (s simpleFixtureBuilder) Fixtures() []*FixtureSet {
	return s.fixtureSets
}

func SimpleFixtureBuilderFactory(fixtureSets []*FixtureSet) FixtureBuilderFactory {
	return func(ctx context.Context) (FixtureBuilder, error) {
		return &simpleFixtureBuilder{
			fixtureSets: fixtureSets,
		}, nil
	}
}
