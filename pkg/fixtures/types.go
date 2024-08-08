package fixtures

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

type FixtureSet interface {
	Write(ctx context.Context) error
}

type SimpleFixtureSet struct {
	Enabled  bool
	Purge    bool
	Writer   FixtureWriter
	Fixtures []interface{}
}

func (c *SimpleFixtureSet) Write(ctx context.Context) error {
	if !c.Enabled {
		return nil
	}

	if c.Writer == nil {
		return fmt.Errorf("fixture set is missing a writer")
	}

	if c.Purge {
		if err := c.Writer.Purge(ctx); err != nil {
			return fmt.Errorf("error during purging of fixture set: %w", err)
		}
	}

	if err := c.Writer.Write(ctx, c.Fixtures); err != nil {
		return fmt.Errorf("error during loading of fixture set: %w", err)
	}

	return nil
}

type FixtureSetFactory func(ctx context.Context, config cfg.Config, logger log.Logger) ([]FixtureSet, error)

//go:generate mockery --name FixtureLoader
type FixtureLoader interface {
	Load(ctx context.Context, fixtureSets []FixtureSet) error
}

//go:generate mockery --name FixtureWriter
type FixtureWriter interface {
	Purge(ctx context.Context) error
	Write(ctx context.Context, fixtures []any) error
}
