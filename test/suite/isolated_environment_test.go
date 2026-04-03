//go:build integration && fixtures

package suite_test

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/fixtures"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/test/suite"
	"github.com/stretchr/testify/assert"
)

// test suite to ensure we load shared fixtures a second time when they are not in a shared environment
type isolatedEnvironmentSuite struct {
	suite.Suite
}

type markerKey struct{}

var isolatedEnvironmentWrites atomic.Int32

type isolatedEnvironmentWriter struct{}

func (s *isolatedEnvironmentSuite) SetupSuite() []suite.Option {
	return []suite.Option{
		suite.WithFixtureSetFactory(fixtures.NewFixtureSetsFactory(func(ctx context.Context, config cfg.Config, logger log.Logger) (fixtures.FixtureSet, error) {
			writer := fixtures.NewManagedFixtureWriter(isolatedEnvironmentWriter{}, "test/shared-fixture-isolated-environment")

			return fixtures.NewSimpleFixtureSet[string](nil, writer, fixtures.WithShared(true), fixtures.WithSharedKey("shared-fixture-isolated-environment")), nil
		})),
	}
}

func (s *isolatedEnvironmentSuite) TestSharedFixtureFirstLoad() {
	s.assertSharedFixtureLoaded()
}

func (s *isolatedEnvironmentSuite) TestSharedFixtureSecondLoad() {
	s.assertSharedFixtureLoaded()
}

func (s *isolatedEnvironmentSuite) assertSharedFixtureLoaded() {
	loaded, err := appctx.Get[bool](s.Env().Context(), markerKey{})
	s.NoError(err)
	s.True(loaded)
}

func (w isolatedEnvironmentWriter) Write(ctx context.Context, _ []any) error {
	isolatedEnvironmentWrites.Add(1)

	_, err := appctx.Provide(ctx, markerKey{}, func() (bool, error) {
		return true, nil
	})

	return err
}

func TestSharedFixtureIsolatedEnvironmentSuite(t *testing.T) {
	isolatedEnvironmentWrites.Store(0)

	suite.Run(t, &isolatedEnvironmentSuite{})

	assert.Equal(t, int32(2), isolatedEnvironmentWrites.Load())
}
