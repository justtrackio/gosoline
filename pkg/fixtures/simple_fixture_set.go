package fixtures

import (
	"context"
	"fmt"
	"strings"

	"github.com/justtrackio/gosoline/pkg/mdl"
)

var (
	_ FixtureSet         = &simpleFixtureSet[struct{}]{}
	_ SharedAware        = &simpleFixtureSet[struct{}]{}
	_ FixtureWriterAware = &simpleFixtureSet[struct{}]{}
)

type simpleFixtureSet[T any] struct {
	fixtures NamedFixtures[T]
	settings FixtureSetSettings
	writer   FixtureWriter
}

func NewSimpleFixtureSet[T any](fixtures NamedFixtures[T], writer FixtureWriter, options ...FixtureSetOption) FixtureSet {
	settings := NewFixtureSetSettings(options...)

	return &simpleFixtureSet[T]{
		fixtures: fixtures,
		settings: *settings,
		writer:   writer,
	}
}

func (c *simpleFixtureSet[T]) Write(ctx context.Context) error {
	if c.writer == nil {
		return fmt.Errorf("fixture set is missing a writer")
	}

	if !c.settings.Enabled {
		return nil
	}

	allFixtures := c.fixtures.All()
	if err := c.writer.Write(ctx, allFixtures); err != nil {
		t := new(T)

		return fmt.Errorf("failed to write fixtures for type %T: %w", *t, err)
	}

	return nil
}

func (c *simpleFixtureSet[T]) String() string {
	var model any = mdl.Empty[T]()

	if c.fixtures.Len() > 0 {
		model = c.fixtures[0].Value
		if kvModel, ok := model.(anyTypedValueAware); ok {
			model = kvModel.GetValue()
		}
	}

	return fmt.Sprintf("%T(len=%d, type=%T)", c, c.fixtures.Len(), model)
}

func (c *simpleFixtureSet[T]) IsShared() bool {
	return c.settings.Shared
}

func (c *simpleFixtureSet[T]) SharedKey() string {
	if c.settings.SharedKey != "" {
		return c.settings.SharedKey
	}

	parts := make([]string, 0, c.fixtures.Len())
	for _, fixture := range c.fixtures {
		parts = append(parts, fixture.Name)
	}

	resourceIds, _ := FixtureSetResourceIds(c)

	return fmt.Sprintf("%T|%v|%s", c, resourceIds, strings.Join(parts, ","))
}

func (c *simpleFixtureSet[T]) FixtureWriter() FixtureWriter {
	return c.writer
}

func (c *simpleFixtureSet[T]) isEnabled() bool {
	return c.settings.Enabled
}

func (c *simpleFixtureSet[T]) IsEnabled() bool {
	return c.isEnabled()
}
