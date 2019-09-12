package kvstore_test

import (
	"context"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/kvstore"
	monMocks "github.com/applike/gosoline/pkg/mon/mocks"
	"github.com/applike/gosoline/pkg/redis/mocks"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRedisKvStore_Contains(t *testing.T) {
	logger := monMocks.NewLoggerMockedAll()

	client := new(mocks.Client)
	client.On("Exists", "applike-gosoline-kvstore-kvstore-test-foo").Return(int64(0), nil)
	client.On("Exists", "applike-gosoline-kvstore-kvstore-test-bar").Return(int64(1), nil)

	store := kvstore.NewRedisKvStoreWithInterfaces(logger, client, &kvstore.Settings{
		AppId: cfg.AppId{
			Project:     "applike",
			Environment: "test",
			Family:      "gosoline",
			Application: "kvstore",
		},
		Name: "test",
	})

	exists, err := store.Contains(context.Background(), "foo")
	assert.NoError(t, err)
	assert.False(t, exists)

	exists, err = store.Contains(context.Background(), "bar")
	assert.NoError(t, err)
	assert.True(t, exists)

	client.AssertExpectations(t)
}
