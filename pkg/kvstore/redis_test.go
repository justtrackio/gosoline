package kvstore_test

import (
	"context"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/kvstore"
	redisMocks "github.com/applike/gosoline/pkg/redis/mocks"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

type Item struct {
	Id   string `json:"id"`
	Body string `json:"body"`
}

func TestRedisKvStore_Contains(t *testing.T) {
	store, client := buildTestableRedisStore()

	client.On("Exists", "applike-gosoline-kvstore-kvstore-test-foo").Return(int64(0), nil)
	client.On("Exists", "applike-gosoline-kvstore-kvstore-test-bar").Return(int64(1), nil)

	exists, err := store.Contains(context.Background(), "foo")
	assert.NoError(t, err)
	assert.False(t, exists)

	exists, err = store.Contains(context.Background(), "bar")
	assert.NoError(t, err)
	assert.True(t, exists)

	client.AssertExpectations(t)
}

func TestRedisKvStore_Get(t *testing.T) {
	store, client := buildTestableRedisStore()
	client.On("Get", "applike-gosoline-kvstore-kvstore-test-foo").Return(`{"id":"foo","body":"bar"}`, nil)

	item := &Item{}
	found, err := store.Get(context.Background(), "foo", item)

	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "foo", item.Id)
	assert.Equal(t, "bar", item.Body)

	client.AssertExpectations(t)
}

func TestRedisKvStore_GetBatch(t *testing.T) {
	store, client := buildTestableRedisStore()

	args := []interface{}{"applike-gosoline-kvstore-kvstore-test-foo", "applike-gosoline-kvstore-kvstore-test-fuu"}
	returns := []interface{}{`{"id":"foo","body":"bar"}`, nil}

	client.On("MGet", args...).Return(returns, nil)

	keys := []string{"foo", "fuu"}
	result := make(map[string]Item)

	missing, err := store.GetBatch(context.Background(), keys, result)

	assert.NoError(t, err)
	assert.Contains(t, result, "foo")
	assert.Equal(t, "foo", result["foo"].Id)
	assert.Equal(t, "bar", result["foo"].Body)

	assert.Len(t, missing, 1)
	assert.Contains(t, missing, "fuu")

	client.AssertExpectations(t)
}

func TestRedisKvStore_Put(t *testing.T) {
	store, client := buildTestableRedisStore()
	client.On("Set", "applike-gosoline-kvstore-kvstore-test-foo", []byte(`{"id":"foo","body":"bar"}`), time.Duration(0)).Return(nil)

	item := &Item{
		Id:   "foo",
		Body: "bar",
	}

	err := store.Put(context.Background(), "foo", item)

	assert.NoError(t, err)
	client.AssertExpectations(t)
}

func TestRedisKvStore_PutBatch(t *testing.T) {
	store, client := buildTestableRedisStore()
	client.On("Set", "applike-gosoline-kvstore-kvstore-test-foo", []byte(`{"id":"foo","body":"bar"}`), time.Duration(0)).Return(nil)
	client.On("Set", "applike-gosoline-kvstore-kvstore-test-fuu", []byte(`{"id":"fuu","body":"baz"}`), time.Duration(0)).Return(nil)

	items := map[string]Item{
		"foo": {
			Id:   "foo",
			Body: "bar",
		},
		"fuu": {
			Id:   "fuu",
			Body: "baz",
		},
	}

	err := store.PutBatch(context.Background(), items)

	assert.NoError(t, err)
	client.AssertExpectations(t)
}

func buildTestableRedisStore() (*kvstore.RedisKvStore, *redisMocks.Client) {
	client := new(redisMocks.Client)

	store := kvstore.NewRedisKvStoreWithInterfaces(client, &kvstore.Settings{
		AppId: cfg.AppId{
			Project:     "applike",
			Environment: "test",
			Family:      "gosoline",
			Application: "kvstore",
		},
		Name:      "test",
		BatchSize: 100,
	})

	return store, client
}
