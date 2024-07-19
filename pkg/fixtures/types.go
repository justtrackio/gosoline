package fixtures

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

type FixtureSet struct {
	Enabled        bool
	Purge          bool
	Writer         FixtureWriterFactory
	Fixtures       []interface{}
	FixtureSetName string
}

type FixtureBuilderFactory func(ctx context.Context, config cfg.Config, logger log.Logger) (FixtureBuilder, error)

//go:generate mockery --name FixtureBuilder
type FixtureBuilder interface {
	Fixtures() []*FixtureSet
}

//go:generate mockery --name FixtureLoader
type FixtureLoader interface {
	Load(ctx context.Context, fixtureSets []*FixtureSet) error
}

//go:generate mockery --name FixtureWriter
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

func SimpleFixtureBuilder(fixtureSets []*FixtureSet) (FixtureBuilder, error) {
	return &simpleFixtureBuilder{
		fixtureSets: fixtureSets,
	}, nil
}

func SimpleFixtureBuilderFactory(fixtureSets []*FixtureSet) FixtureBuilderFactory {
	return func(_ context.Context, _ cfg.Config, _ log.Logger) (FixtureBuilder, error) {
		return SimpleFixtureBuilder(fixtureSets)
	}
}
