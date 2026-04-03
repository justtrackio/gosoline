package reslife

import (
	"context"
	"testing"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	reslifeMocks "github.com/justtrackio/gosoline/pkg/reslife/mocks"
	"github.com/justtrackio/gosoline/pkg/test/matcher"
	"github.com/stretchr/testify/require"
)

type testPurgeResource struct {
	*reslifeMocks.Purger
	id string
}

func (t *testPurgeResource) GetId() string {
	return t.id
}

func TestLifeCycleManagerPurgeSelected(t *testing.T) {
	ctx := appctx.WithContainer(context.Background())
	config := cfg.New()
	require.NoError(t, config.Option(cfg.WithConfigMap(map[string]any{
		"resource_lifecycles": map[string]any{
			"purge": map[string]any{
				"enabled": true,
			},
		},
	})))

	resourceA := &testPurgeResource{Purger: reslifeMocks.NewPurger(t), id: "redis/a"}
	resourceB := &testPurgeResource{Purger: reslifeMocks.NewPurger(t), id: "redis/b"}
	resourceB.EXPECT().Purge(matcher.Context).Return(nil).Once()
	require.NoError(t, AddLifeCycleer(ctx, func(ctx context.Context, config cfg.Config, logger log.Logger) (LifeCycleer, error) {
		return resourceA, nil
	}))
	require.NoError(t, AddLifeCycleer(ctx, func(ctx context.Context, config cfg.Config, logger log.Logger) (LifeCycleer, error) {
		return resourceB, nil
	}))

	logger := logMocks.NewLogger(t)
	logger.EXPECT().WithChannel("lifecycle-manager").Return(log.NewLogger()).Once()

	manager, err := NewLifeCycleManager(ctx, config, logger)
	require.NoError(t, err)

	require.NoError(t, manager.PurgeSelected(ctx, []string{"redis/b"}))
}
