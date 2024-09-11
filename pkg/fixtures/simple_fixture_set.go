package fixtures

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/mdl"
)

type simpleFixtureSet[T any] struct {
	Enabled  bool
	Purge    bool
	Writer   FixtureWriter
	Fixtures NamedFixtures[T]
}

func NewSimpleFixtureSet[T any](fixtures NamedFixtures[T], writer FixtureWriter, options ...FixtureSetOption) FixtureSet {
	settings := NewFixtureSetSettings(options...)

	return &simpleFixtureSet[T]{
		Fixtures: fixtures,
		Writer:   writer,
		Enabled:  settings.Enabled,
		Purge:    settings.Purge,
	}
}

func (c *simpleFixtureSet[T]) Write(ctx context.Context) error {
	if c.Writer == nil {
		return fmt.Errorf("fixture set is missing a writer")
	}

	if !c.Enabled {
		return nil
	}

	if c.Purge {
		if err := c.Writer.Purge(ctx); err != nil {
			return fmt.Errorf("error during purging of fixture set: %w", err)
		}
	}

	allFixtures := c.Fixtures.All()
	if err := c.Writer.Write(ctx, allFixtures); err != nil {
		t := new(T)

		return fmt.Errorf("failed to write fixtures for type %T: %w", *t, err)
	}

	return nil
}

func (c *simpleFixtureSet[T]) String() string {
	var model any = mdl.Empty[T]()

	if c.Fixtures.Len() > 0 {
		model = c.Fixtures[0].Value
		if kvModel, ok := model.(*KvStoreFixture); ok {
			model = kvModel.Value
		}
	}

	return fmt.Sprintf("%T(len=%d, type=%T)", c, c.Fixtures.Len(), model)
}
