//go:build fixtures

package fixtures

import (
	"context"
	"testing"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	gosoFixtures "github.com/justtrackio/gosoline/pkg/fixtures"
	fixtureMocks "github.com/justtrackio/gosoline/pkg/fixtures/mocks"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockedFixtureSet struct {
	*fixtureMocks.FixtureSet
	*fixtureMocks.SharedAware
	writer gosoFixtures.FixtureWriter
}

func newMockedFixtureSet(t *testing.T, writer gosoFixtures.FixtureWriter, shared bool, sharedKey string) *mockedFixtureSet {
	fixtureSet := &mockedFixtureSet{
		FixtureSet:  fixtureMocks.NewFixtureSet(t),
		SharedAware: fixtureMocks.NewSharedAware(t),
		writer:      writer,
	}

	fixtureSet.SharedAware.EXPECT().IsShared().Return(shared).Maybe()
	fixtureSet.SharedAware.EXPECT().SharedKey().Return(sharedKey).Maybe()

	return fixtureSet
}

func (m *mockedFixtureSet) FixtureWriter() gosoFixtures.FixtureWriter {
	return m.writer
}

func TestFixtureLoaderSkipsAlreadyLoadedSharedFixtureSets(t *testing.T) {
	ctx := appctx.WithContainer(context.Background())
	config := cfg.New()
	require.NoError(t, config.Option(cfg.WithConfigMap(map[string]any{
		"fixtures": map[string]any{
			"enabled": true,
			"groups":  []string{"default"},
		},
	})))
	logger := log.NewLogger()
	sharedFixtureSet := newMockedFixtureSet(t, gosoFixtures.NewManagedFixtureWriter(fixtureMocks.NewFixtureWriter(t), "redis/shared"), true, "shared-set")
	mutableFixtureSet := newMockedFixtureSet(t, gosoFixtures.NewManagedFixtureWriter(fixtureMocks.NewFixtureWriter(t), "redis/mutable"), false, "")
	sharedFixtureSet.FixtureSet.EXPECT().Write(ctx).Return(nil).Once()
	mutableFixtureSet.FixtureSet.EXPECT().Write(ctx).Return(nil).Twice()

	loader, err := gosoFixtures.NewFixtureLoader(ctx, config, logger, map[string][]gosoFixtures.FixtureSet{
		"default": {sharedFixtureSet, mutableFixtureSet},
	}, nil)
	require.NoError(t, err)

	require.NoError(t, loader.Load(ctx))
	require.NoError(t, loader.Load(ctx))

	resourceIds, ok := gosoFixtures.MutableResourceIds(loader)
	require.True(t, ok)
	assert.Equal(t, []string{"redis/mutable"}, resourceIds)
}

func TestFixtureLoaderRejectsSharedFixtureSetWithoutResourceIds(t *testing.T) {
	ctx := appctx.WithContainer(context.Background())
	config := cfg.New()
	require.NoError(t, config.Option(cfg.WithConfigMap(map[string]any{
		"fixtures": map[string]any{
			"enabled": true,
			"groups":  []string{"default"},
		},
	})))

	_, err := gosoFixtures.NewFixtureLoader(ctx, config, log.NewLogger(), map[string][]gosoFixtures.FixtureSet{
		"default": {
			newMockedFixtureSet(t, fixtureMocks.NewFixtureWriter(t), true, "shared-set"),
		},
	}, nil)
	require.Error(t, err)
	assert.ErrorContains(t, err, "shared fixture set")
	assert.ErrorContains(t, err, "missing managed resources on its writer")
}

func TestFixtureLoaderRejectsMutableFixtureSetWithoutResourceIdsWhenSharedFixturesExist(t *testing.T) {
	ctx := appctx.WithContainer(context.Background())
	config := cfg.New()
	require.NoError(t, config.Option(cfg.WithConfigMap(map[string]any{
		"fixtures": map[string]any{
			"enabled": true,
			"groups":  []string{"default"},
		},
	})))

	_, err := gosoFixtures.NewFixtureLoader(ctx, config, log.NewLogger(), map[string][]gosoFixtures.FixtureSet{
		"default": {
			newMockedFixtureSet(t, gosoFixtures.NewManagedFixtureWriter(fixtureMocks.NewFixtureWriter(t), "redis/shared"), true, "shared-set"),
			newMockedFixtureSet(t, fixtureMocks.NewFixtureWriter(t), false, ""),
		},
	}, nil)
	require.Error(t, err)
	assert.ErrorContains(t, err, "missing managed resources on its writer while shared fixtures are enabled")
	assert.ErrorContains(t, err, "fixtures.NewUnmanagedFixtureWriter(...)")
}

func TestFixtureLoaderAllowsUnmanagedWriterWhenSharedFixturesExist(t *testing.T) {
	ctx := appctx.WithContainer(context.Background())
	config := cfg.New()
	require.NoError(t, config.Option(cfg.WithConfigMap(map[string]any{
		"fixtures": map[string]any{
			"enabled": true,
			"groups":  []string{"default"},
		},
	})))

	_, err := gosoFixtures.NewFixtureLoader(ctx, config, log.NewLogger(), map[string][]gosoFixtures.FixtureSet{
		"default": {
			newMockedFixtureSet(t, gosoFixtures.NewManagedFixtureWriter(fixtureMocks.NewFixtureWriter(t), "redis/shared"), true, "shared-set"),
			newMockedFixtureSet(t, gosoFixtures.NewUnmanagedFixtureWriter(fixtureMocks.NewFixtureWriter(t)), false, ""),
		},
	}, nil)
	require.NoError(t, err)
}

func TestFixtureLoaderRejectsSharedFixtureSetWithUnmanagedWriter(t *testing.T) {
	ctx := appctx.WithContainer(context.Background())
	config := cfg.New()
	require.NoError(t, config.Option(cfg.WithConfigMap(map[string]any{
		"fixtures": map[string]any{
			"enabled": true,
			"groups":  []string{"default"},
		},
	})))

	_, err := gosoFixtures.NewFixtureLoader(ctx, config, log.NewLogger(), map[string][]gosoFixtures.FixtureSet{
		"default": {
			newMockedFixtureSet(t, gosoFixtures.NewUnmanagedFixtureWriter(fixtureMocks.NewFixtureWriter(t)), true, "shared-set"),
		},
	}, nil)
	require.Error(t, err)
	assert.ErrorContains(t, err, "uses a writer without managed resources")
}

func TestFixtureLoaderKeepsBackwardCompatibilityWithoutSharedFixtures(t *testing.T) {
	ctx := appctx.WithContainer(context.Background())
	config := cfg.New()
	require.NoError(t, config.Option(cfg.WithConfigMap(map[string]any{
		"fixtures": map[string]any{
			"enabled": true,
			"groups":  []string{"default"},
		},
	})))

	_, err := gosoFixtures.NewFixtureLoader(ctx, config, log.NewLogger(), map[string][]gosoFixtures.FixtureSet{
		"default": {
			newMockedFixtureSet(t, fixtureMocks.NewFixtureWriter(t), false, ""),
		},
	}, nil)
	require.NoError(t, err)
}
