//go:build integration && fixtures

// snippet-start: imports
package apitest

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/fixtures"
	"github.com/justtrackio/gosoline/pkg/kvstore"
	"github.com/justtrackio/gosoline/pkg/log"
)

// snippet-end: imports

// snippet-start: fixtures
var namedFixtures = fixtures.NamedFixtures[*kvstore.KvStoreFixture]{
	{
		Name: "Currency_current_GBP",
		Value: &kvstore.KvStoreFixture{
			Key:   "GBP",
			Value: 1.25,
		},
	},
	{
		Name: "Currency_old_GBP",
		Value: &kvstore.KvStoreFixture{
			Key:   "2021-01-03-GBP",
			Value: 0.8,
		},
	},
}

// snippet-end: fixtures

// snippet-start: fixture sets factory
func fixtureSetsFactory(ctx context.Context, config cfg.Config, logger log.Logger, group string) ([]fixtures.FixtureSet, error) {
	writer, err := kvstore.NewConfigurableKvStoreFixtureWriter[float64](ctx, config, logger, "currency")
	if err != nil {
		return nil, fmt.Errorf("failed to create kvstore fixture writer: %w", err)
	}

	sfs := fixtures.NewSimpleFixtureSet(namedFixtures, writer)

	return []fixtures.FixtureSet{
		sfs,
	}, nil
}

// snippet-end: fixture sets factory
