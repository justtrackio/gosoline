package fixtures

import (
	"context"
	"fmt"
	"strings"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

type (
	FixtureSetFactory  func(ctx context.Context, config cfg.Config, logger log.Logger) (FixtureSet, error)
	FixtureSetsFactory func(ctx context.Context, config cfg.Config, logger log.Logger, group string) ([]FixtureSet, error)
)

// SharedAware is implemented by fixture sets that can be reused within a shared
// test environment.
//
//go:generate go run github.com/vektra/mockery/v2 --name SharedAware
type SharedAware interface {
	IsShared() bool
	SharedKey() string
}

// ResourceAware is implemented by fixture writers that can describe which
// lifecycle-managed resources they mutate.
//
//go:generate go run github.com/vektra/mockery/v2 --name ResourceAware
type ResourceAware interface {
	HasManagedResources() bool
	ManagedResourceIds() []string
}

// FixtureWriterAware is implemented by fixture sets that expose the writer used
// to persist their data.
type FixtureWriterAware interface {
	FixtureWriter() FixtureWriter
}

// EnabledAware is implemented by fixture sets that can report whether they are
// active for the current load.
type EnabledAware interface {
	IsEnabled() bool
}

//go:generate go run github.com/vektra/mockery/v2 --name MutableResourceAware
type MutableResourceAware interface {
	MutableResourceIds() ([]string, bool)
}

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

func IsSharedFixtureSet(fixtureSet FixtureSet) bool {
	sharedAware, ok := fixtureSet.(SharedAware)
	if !ok {
		return false
	}

	return sharedAware.IsShared()
}

func SharedFixtureSetKey(fixtureSet FixtureSet) string {
	sharedAware, ok := fixtureSet.(SharedAware)
	if !ok {
		return ""
	}

	return sharedAware.SharedKey()
}

func FixtureSetResourceIds(fixtureSet FixtureSet) ([]string, bool) {
	writerAware, ok := fixtureSet.(FixtureWriterAware)
	if !ok {
		return nil, false
	}

	resourceAware, ok := writerAware.FixtureWriter().(ResourceAware)
	if !ok {
		return nil, false
	}

	if !resourceAware.HasManagedResources() {
		return nil, false
	}

	return resourceAware.ManagedResourceIds(), true
}

func FixtureSetHasNoManagedResources(fixtureSet FixtureSet) bool {
	writerAware, ok := fixtureSet.(FixtureWriterAware)
	if !ok {
		return false
	}

	resourceAware, ok := writerAware.FixtureWriter().(ResourceAware)
	if !ok {
		return false
	}

	return !resourceAware.HasManagedResources()
}

func FixtureSetDescription(fixtureSet FixtureSet) string {
	sharedSuffix := ""
	if IsSharedFixtureSet(fixtureSet) {
		sharedSuffix = " (shared)"
	}

	resourceIds, hasResourceIds := FixtureSetResourceIds(fixtureSet)
	if hasResourceIds {
		return fmt.Sprintf("%T%s resources=[%s]", fixtureSet, sharedSuffix, strings.Join(resourceIds, ", "))
	}

	if FixtureSetHasNoManagedResources(fixtureSet) {
		return fmt.Sprintf("%T%s resources=[none]", fixtureSet, sharedSuffix)
	}

	return fmt.Sprintf("%T%s", fixtureSet, sharedSuffix)
}

func MutableResourceIds(loader FixtureLoader) ([]string, bool) {
	typedLoader, ok := loader.(MutableResourceAware)
	if !ok {
		return nil, false
	}

	return typedLoader.MutableResourceIds()
}
