package fixtures

import (
	"slices"

	"github.com/justtrackio/gosoline/pkg/funk"
)

var (
	_ ResourceAware = managedFixtureWriter{}
	_ ResourceAware = unmanagedFixtureWriter{}
)

type managedFixtureWriter struct {
	FixtureWriter
	resourceIds []string
}

type unmanagedFixtureWriter struct {
	FixtureWriter
}

func NewManagedFixtureWriter(writer FixtureWriter, resourceIds ...string) FixtureWriter {
	if writer == nil {
		return nil
	}

	return managedFixtureWriter{
		FixtureWriter: writer,
		resourceIds: funk.Filter(funk.Uniq(resourceIds), func(s string) bool {
			return s != ""
		}),
	}
}

func NewUnmanagedFixtureWriter(writer FixtureWriter) FixtureWriter {
	if writer == nil {
		return nil
	}

	return unmanagedFixtureWriter{FixtureWriter: writer}
}

func (m managedFixtureWriter) HasManagedResources() bool {
	return true
}

func (m managedFixtureWriter) ManagedResourceIds() []string {
	return slices.Clone(m.resourceIds)
}

func (u unmanagedFixtureWriter) HasManagedResources() bool {
	return false
}

func (u unmanagedFixtureWriter) ManagedResourceIds() []string {
	return nil
}
