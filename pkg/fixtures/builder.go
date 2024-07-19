package fixtures

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

// FixtureSetBuilder gets the whole set of fixtures (slice of maps to any) by their model names (root map keys)
// to create fixtureSets out of them.
// A FixtureSetBuilder may remove any data the application does not need as fixtures and transform them in any needed way.
// For example there may be fixtures originally for a db, however the currently requesting application may transform them
// into ddb fixtures.

// ModelBasedFixtures example data:
//
//	map[string][]map[string]any{
//	  "users": {
//	    {
//	      "username": "alice",
//	    },
//	  },
//	}
type ModelBasedFixtures map[string][]map[string]any

type FixtureSetBuilderSettings struct {
	DbName   string
	Fixtures ModelBasedFixtures
	Enabled  bool
	Purge    bool
}

func NewFixtureSetBuilderSettings(dbName string, fixtures ModelBasedFixtures, enabled bool, purge bool) FixtureSetBuilderSettings {
	return FixtureSetBuilderSettings{
		DbName:   dbName,
		Fixtures: fixtures,
		Enabled:  enabled,
		Purge:    purge,
	}
}

type FixtureSetBuilder func(ctx context.Context, config cfg.Config, logger log.Logger, settings FixtureSetBuilderSettings) ([]*FixtureSet, error)

// FixtureSetBuilders stores converters for model fixtures to fixtureSets
var FixtureSetBuilders = map[string]FixtureSetBuilder{}

func WithConverter(name string, converter FixtureSetBuilder) {
	FixtureSetBuilders[name] = converter
}
