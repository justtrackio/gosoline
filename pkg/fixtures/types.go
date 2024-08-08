package fixtures

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

type FixtureSet interface {
	Write(ctx context.Context) error
}

type CodeFixtureSet struct {
	Enabled  bool
	Purge    bool
	Writer   FixtureWriter
	Fixtures []interface{}
}

func (c CodeFixtureSet) Write(ctx context.Context) error {
	panic("implement me")
}

type FixtureSetFactory func(ctx context.Context, config cfg.Config, logger log.Logger) ([]FixtureSet, error)

//go:generate mockery --name FixtureLoader
type FixtureLoader interface {
	Load(ctx context.Context, fixtureSets []FixtureSet) error
}

//go:generate mockery --name FixtureWriter
type FixtureWriter interface {
	Purge(ctx context.Context) error
	Write(ctx context.Context, fixture *FixtureSet) error
}

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

func SimpleFixtureBuilderFactory(fixtureSets []*FixtureSet) FixtureSetFactory {
	return func(_ context.Context, _ cfg.Config, _ log.Logger) (FixtureBuilder, error) {
		return SimpleFixtureBuilder(fixtureSets)
	}
}
