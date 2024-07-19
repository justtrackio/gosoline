package fixtures

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

type (
	FixtureSetFactory  func(ctx context.Context, config cfg.Config, logger log.Logger) (FixtureSet, error)
	FixtureSetsFactory func(ctx context.Context, config cfg.Config, logger log.Logger) ([]FixtureSet, error)
)

//go:generate mockery --name FixtureLoader
type FixtureLoader interface {
	Load(ctx context.Context, fixtureSets []FixtureSet) error
}

//go:generate mockery --name FixtureSet
type FixtureSet interface {
	Write(ctx context.Context) error
}

//go:generate mockery --name FixtureWriter
type FixtureWriter interface {
	Purge(ctx context.Context) error
	Write(ctx context.Context, fixtures []any) error
}

func NewFixtureSetsFactory(factories ...FixtureSetFactory) FixtureSetsFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) ([]FixtureSet, error) {
		var err error
		var set FixtureSet
		var sets []FixtureSet

		for _, factory := range factories {
			if set, err = factory(ctx, config, logger); err != nil {
				return nil, err
			}

			sets = append(sets, set)
		}

		return sets, nil
	}
}
