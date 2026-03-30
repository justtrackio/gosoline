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

// test suite to ensure we load shared fixtures once and mutable fixtures for every test
type sharedEnvironmentSuite struct {
	suite.Suite
}

type (
	sharedMarkerKey  struct{}
	mutableMarkerKey struct{}
)

var (
	sharedEnvironmentSharedWrites  atomic.Int32
	sharedEnvironmentMutableWrites atomic.Int32
)

type (
	sharedEnvironmentSharedWriter  struct{}
	sharedEnvironmentMutableWriter struct{}
)

func (s *sharedEnvironmentSuite) SetupSuite() []suite.Option {
	return []suite.Option{
		suite.WithSharedEnvironment(),
		suite.WithFixtureSetFactory(fixtures.NewFixtureSetsFactory(
			func(ctx context.Context, config cfg.Config, logger log.Logger) (fixtures.FixtureSet, error) {
				writer := fixtures.NewManagedFixtureWriter(sharedEnvironmentSharedWriter{}, "test/shared-environment/shared")

				return fixtures.NewSimpleFixtureSet[string](nil, writer, fixtures.WithShared(true), fixtures.WithSharedKey("shared-environment-shared")), nil
			},
			func(ctx context.Context, config cfg.Config, logger log.Logger) (fixtures.FixtureSet, error) {
				writer := fixtures.NewManagedFixtureWriter(sharedEnvironmentMutableWriter{}, "test/shared-environment/mutable")

				return fixtures.NewSimpleFixtureSet[string](nil, writer), nil
			},
		)),
	}
}

func (s *sharedEnvironmentSuite) TestSharedFixtureFirstLoad() {
	s.assertFixturesLoaded()
}

func (s *sharedEnvironmentSuite) TestSharedFixtureSecondLoad() {
	s.assertFixturesLoaded()
}

func (s *sharedEnvironmentSuite) assertFixturesLoaded() {
	sharedLoaded, err := appctx.Get[bool](s.Env().Context(), sharedMarkerKey{})
	s.NoError(err)
	s.True(sharedLoaded)

	mutableLoaded, err := appctx.Get[bool](s.Env().Context(), mutableMarkerKey{})
	s.NoError(err)
	s.True(mutableLoaded)
}

func (w sharedEnvironmentSharedWriter) Write(ctx context.Context, _ []any) error {
	sharedEnvironmentSharedWrites.Add(1)

	_, err := appctx.Provide(ctx, sharedMarkerKey{}, func() (bool, error) {
		return true, nil
	})

	return err
}

func (w sharedEnvironmentMutableWriter) Write(ctx context.Context, _ []any) error {
	sharedEnvironmentMutableWrites.Add(1)

	_, err := appctx.Provide(ctx, mutableMarkerKey{}, func() (bool, error) {
		return true, nil
	})

	return err
}

func TestSharedFixtureSharedEnvironmentSuite(t *testing.T) {
	sharedEnvironmentSharedWrites.Store(0)
	sharedEnvironmentMutableWrites.Store(0)

	suite.Run(t, &sharedEnvironmentSuite{})

	assert.Equal(t, int32(1), sharedEnvironmentSharedWrites.Load())
	assert.Equal(t, int32(2), sharedEnvironmentMutableWrites.Load())
}
