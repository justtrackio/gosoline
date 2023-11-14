package kvstore_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kvstore"
	"github.com/justtrackio/gosoline/pkg/mdl"
	redisMocks "github.com/justtrackio/gosoline/pkg/redis/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type Item struct {
	Id   string `json:"id"`
	Body string `json:"body"`
}

func TestRedisKvStore_Contains(t *testing.T) {
	ctx, store, client := buildTestableRedisStore[Item]()

	client.On("Exists", ctx, "justtrack-gosoline-grp-kvstore-test-foo").Return(int64(0), nil)
	client.On("Exists", ctx, "justtrack-gosoline-grp-kvstore-test-bar").Return(int64(1), nil)

	exists, err := store.Contains(ctx, "foo")
	assert.NoError(t, err)
	assert.False(t, exists)

	exists, err = store.Contains(ctx, "bar")
	assert.NoError(t, err)
	assert.True(t, exists)

	client.AssertExpectations(t)
}

func TestRedisKvStore_Get(t *testing.T) {
	ctx, store, client := buildTestableRedisStore[Item]()
	client.On("Get", ctx, "justtrack-gosoline-grp-kvstore-test-foo").Return(`{"id":"foo","body":"bar"}`, nil)

	item := &Item{}
	found, err := store.Get(ctx, "foo", item)

	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "foo", item.Id)
	assert.Equal(t, "bar", item.Body)

	client.AssertExpectations(t)
}

func TestRedisKvStore_GetBatch(t *testing.T) {
	ctx, store, client := buildTestableRedisStore[Item]()

	args := []interface{}{ctx, "justtrack-gosoline-grp-kvstore-test-foo", "justtrack-gosoline-grp-kvstore-test-fuu"}
	returns := []interface{}{`{"id":"foo","body":"bar"}`, nil}

	client.On("MGet", args...).Return(returns, nil)

	keys := []string{"foo", "fuu"}
	result := make(map[string]Item)

	missing, err := store.GetBatch(ctx, keys, result)

	assert.NoError(t, err)
	assert.Contains(t, result, "foo")
	assert.Equal(t, "foo", result["foo"].Id)
	assert.Equal(t, "bar", result["foo"].Body)

	assert.Len(t, missing, 1)
	assert.Contains(t, missing, "fuu")

	client.AssertExpectations(t)
}

func TestRedisKvStore_Put(t *testing.T) {
	ctx, store, client := buildTestableRedisStore[Item]()
	client.On("Set", ctx, "justtrack-gosoline-grp-kvstore-test-foo", []byte(`{"id":"foo","body":"bar"}`), time.Duration(0)).Return(nil)

	item := Item{
		Id:   "foo",
		Body: "bar",
	}

	err := store.Put(ctx, "foo", item)

	assert.NoError(t, err)
	client.AssertExpectations(t)
}

func TestRedisKvStore_PutBatch(t *testing.T) {
	ctx, store, client := buildTestableRedisStoreWithTTL[Item]()

	pipe := &redisMocks.Pipeliner{}
	pipe.On("MSet", ctx, mock.MatchedBy(func(input []interface{}) bool {
		possibleInput1 := `[justtrack-gosoline-grp-kvstore-test-foo {"id":"foo","body":"bar"} justtrack-gosoline-grp-kvstore-test-fuu {"id":"fuu","body":"baz"}]`
		possibleInput2 := `[justtrack-gosoline-grp-kvstore-test-fuu {"id":"fuu","body":"baz"} justtrack-gosoline-grp-kvstore-test-foo {"id":"foo","body":"bar"}]`

		inputStr := fmt.Sprintf("%s", input)
		return inputStr == possibleInput1 || inputStr == possibleInput2
	})).Return(nil)
	client.On("Pipeline").Return(pipe)
	pipe.On("TxPipeline").Return(pipe)
	pipe.On("Expire", ctx, "justtrack-gosoline-grp-kvstore-test-foo", mock.AnythingOfType("time.Duration")).Return(nil)
	pipe.On("Expire", ctx, "justtrack-gosoline-grp-kvstore-test-fuu", mock.AnythingOfType("time.Duration")).Return(nil)
	pipe.On("Exec", ctx).Return(nil, nil)

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

	err := store.PutBatch(ctx, items)

	assert.NoError(t, err)
	client.AssertExpectations(t)
}

func TestRedisKvStore_PutBatchSkipExpire(t *testing.T) {
	ctx, store, client := buildTestableRedisStore[Item]()

	pipe := &redisMocks.Pipeliner{}
	pipe.On("MSet", ctx, mock.MatchedBy(func(input []interface{}) bool {
		possibleInput1 := `[justtrack-gosoline-grp-kvstore-test-foo {"id":"foo","body":"bar"} justtrack-gosoline-grp-kvstore-test-fuu {"id":"fuu","body":"baz"}]`
		possibleInput2 := `[justtrack-gosoline-grp-kvstore-test-fuu {"id":"fuu","body":"baz"} justtrack-gosoline-grp-kvstore-test-foo {"id":"foo","body":"bar"}]`

		inputStr := fmt.Sprintf("%s", input)
		return inputStr == possibleInput1 || inputStr == possibleInput2
	})).Return(nil)
	client.On("Pipeline").Return(pipe)
	pipe.On("TxPipeline").Return(pipe)
	pipe.On("Exec", ctx).Return(nil, nil)

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

	err := store.PutBatch(ctx, items)

	assert.NoError(t, err)
	client.AssertExpectations(t)
}

func TestRedisKvStore_EstimateSize(t *testing.T) {
	ctx, store, client := buildTestableRedisStore[Item]()
	client.On("DBSize", ctx).Return(int64(42), nil)

	size := store.(kvstore.SizedStore[Item]).EstimateSize()

	assert.Equal(t, mdl.Box(int64(42)), size)
	client.AssertExpectations(t)
}

func TestRedisKvStore_Delete(t *testing.T) {
	ctx, store, client := buildTestableRedisStore[Item]()
	client.On("Del", ctx, "justtrack-gosoline-grp-kvstore-test-foo").Return(int64(1), nil)

	err := store.Delete(ctx, "foo")

	assert.NoError(t, err)
	client.AssertExpectations(t)
}

func TestRedisKvStore_DeleteBatch(t *testing.T) {
	ctx, store, client := buildTestableRedisStore[Item]()
	client.On("Del", ctx, "justtrack-gosoline-grp-kvstore-test-foo", "justtrack-gosoline-grp-kvstore-test-fuu").Return(int64(2), nil)

	items := []string{"foo", "fuu"}

	err := store.DeleteBatch(ctx, items)

	assert.NoError(t, err)
	client.AssertExpectations(t)
}

func buildTestableRedisStore[T any]() (context.Context, kvstore.KvStore[T], *redisMocks.Client) {
	ctx := context.Background()
	client := new(redisMocks.Client)

	store := kvstore.NewRedisKvStoreWithInterfaces[T](client, &kvstore.Settings{
		AppId: cfg.AppId{
			Project:     "justtrack",
			Environment: "env",
			Family:      "gosoline",
			Group:       "grp",
			Application: "app",
		},
		Name:           "test",
		BatchSize:      100,
		MetricsEnabled: false,
	})

	return ctx, store, client
}

func buildTestableRedisStoreWithTTL[T any]() (context.Context, kvstore.KvStore[T], *redisMocks.Client) {
	ctx := context.Background()
	client := new(redisMocks.Client)

	store := kvstore.NewRedisKvStoreWithInterfaces[T](client, &kvstore.Settings{
		AppId: cfg.AppId{
			Project:     "justtrack",
			Environment: "test",
			Family:      "gosoline",
			Group:       "grp",
			Application: "kvstore",
		},
		Name:           "test",
		BatchSize:      100,
		MetricsEnabled: false,
		Ttl:            time.Second,
	})

	return ctx, store, client
}
