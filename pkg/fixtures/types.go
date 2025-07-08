package fixtures

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

type (
	FixtureSetFactory  func(ctx context.Context, config cfg.Config, logger log.Logger) (FixtureSet, error)
	FixtureSetsFactory func(ctx context.Context, config cfg.Config, logger log.Logger, group string) ([]FixtureSet, error)
)

//go:generate go run github.com/vektra/mockery/v2 --name FixtureLoader
type FixtureLoader interface {
	Load(ctx context.Context) error
}

//go:generate go run github.com/vektra/mockery/v2 --name FixtureSet
type FixtureSet interface {
	Write(ctx context.Context) error
}

//go:generate go run github.com/vektra/mockery/v2 --name FixtureWriter
type FixtureWriter interface {
	Write(ctx context.Context, fixtures []any) error
}

func NewFixtureSetsFactory(factories ...FixtureSetFactory) FixtureSetsFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger, group string) ([]FixtureSet, error) {
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
