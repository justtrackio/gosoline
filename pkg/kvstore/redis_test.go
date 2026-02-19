package kvstore_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/kvstore"
	"github.com/justtrackio/gosoline/pkg/mdl"
	redisMocks "github.com/justtrackio/gosoline/pkg/redis/mocks"
	"github.com/justtrackio/gosoline/pkg/test/matcher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type Item struct {
	Id   string `json:"id"`
	Body string `json:"body"`
}

func TestRedisKvStore_Contains(t *testing.T) {
	ctx, store, client := buildTestableRedisStore[Item](t)

	client.EXPECT().Exists(ctx, "foo").Return(int64(0), nil)
	client.EXPECT().Exists(ctx, "bar").Return(int64(1), nil)

	exists, err := store.Contains(ctx, "foo")
	assert.NoError(t, err)
	assert.False(t, exists)

	exists, err = store.Contains(ctx, "bar")
	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestRedisKvStore_Get(t *testing.T) {
	ctx, store, client := buildTestableRedisStore[Item](t)
	client.EXPECT().Get(ctx, "foo").Return(`{"id":"foo","body":"bar"}`, nil)

	item := &Item{}
	found, err := store.Get(ctx, "foo", item)

	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "foo", item.Id)
	assert.Equal(t, "bar", item.Body)
}

func TestRedisKvStore_GetBatch(t *testing.T) {
	ctx, store, client := buildTestableRedisStore[Item](t)

	args := []any{"foo", "fuu"}
	returns := []any{`{"id":"foo","body":"bar"}`, nil}

	client.EXPECT().MGet(ctx, args...).Return(returns, nil)

	keys := []string{"foo", "fuu"}
	result := make(map[string]Item)

	missing, err := store.GetBatch(ctx, keys, result)

	assert.NoError(t, err)
	assert.Contains(t, result, "foo")
	assert.Equal(t, "foo", result["foo"].Id)
	assert.Equal(t, "bar", result["foo"].Body)

	assert.Len(t, missing, 1)
	assert.Contains(t, missing, "fuu")
}

func TestRedisKvStore_Put(t *testing.T) {
	ctx, store, client := buildTestableRedisStore[Item](t)
	client.EXPECT().Set(ctx, "foo", []byte(`{"id":"foo","body":"bar"}`), time.Duration(0)).Return(nil)

	item := Item{
		Id:   "foo",
		Body: "bar",
	}

	err := store.Put(ctx, "foo", item)

	assert.NoError(t, err)
}

func TestRedisKvStore_PutBatch(t *testing.T) {
	ctx, store, client := buildTestableRedisStoreWithTTL[Item](t)

	pipe := &redisMocks.Pipeliner{}
	pipe.EXPECT().MSet(
		ctx,
		mock.MatchedBy(func(input string) bool {
			return input == "foo" || input == "fuu"
		}),
		mock.MatchedBy(func(input []byte) bool {
			return bytes.Equal(input, []byte(`{"id":"foo","body":"bar"}`)) || bytes.Equal(input, []byte(`{"id":"fuu","body":"baz"}`))
		}),
		mock.MatchedBy(func(input string) bool {
			return input == "foo" || input == "fuu"
		}),
		mock.MatchedBy(func(input []byte) bool {
			return bytes.Equal(input, []byte(`{"id":"foo","body":"bar"}`)) || bytes.Equal(input, []byte(`{"id":"fuu","body":"baz"}`))
		}),
	).Return(nil)

	client.EXPECT().Pipeline().Return(pipe)
	pipe.EXPECT().TxPipeline().Return(pipe)
	pipe.EXPECT().Expire(ctx, "foo", mock.AnythingOfType("time.Duration")).Return(nil)
	pipe.EXPECT().Expire(ctx, "fuu", mock.AnythingOfType("time.Duration")).Return(nil)
	pipe.EXPECT().Exec(ctx).Return(nil, nil)

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
}

func TestRedisKvStore_PutBatchSkipExpire(t *testing.T) {
	ctx, store, client := buildTestableRedisStore[Item](t)

	pipe := &redisMocks.Pipeliner{}
	pipe.EXPECT().MSet(
		ctx,
		mock.MatchedBy(func(input string) bool {
			return input == "foo" || input == "fuu"
		}),
		mock.MatchedBy(func(input []byte) bool {
			return bytes.Equal(input, []byte(`{"id":"foo","body":"bar"}`)) || bytes.Equal(input, []byte(`{"id":"fuu","body":"baz"}`))
		}),
		mock.MatchedBy(func(input string) bool {
			return input == "foo" || input == "fuu"
		}),
		mock.MatchedBy(func(input []byte) bool {
			return bytes.Equal(input, []byte(`{"id":"foo","body":"bar"}`)) || bytes.Equal(input, []byte(`{"id":"fuu","body":"baz"}`))
		}),
	).Return(nil)

	client.EXPECT().Pipeline().Return(pipe)
	pipe.EXPECT().TxPipeline().Return(pipe)
	pipe.EXPECT().Exec(ctx).Return(nil, nil)

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
}

func TestRedisKvStore_EstimateSize(t *testing.T) {
	_, store, client := buildTestableRedisStore[Item](t)
	client.EXPECT().DBSize(matcher.Context).Return(int64(42), nil)

	size := store.(kvstore.SizedStore[Item]).EstimateSize()

	assert.Equal(t, mdl.Box(int64(42)), size)
}

func TestRedisKvStore_Delete(t *testing.T) {
	ctx, store, client := buildTestableRedisStore[Item](t)
	client.EXPECT().Del(ctx, "foo").Return(int64(1), nil)

	err := store.Delete(ctx, "foo")

	assert.NoError(t, err)
}

func TestRedisKvStore_DeleteBatch(t *testing.T) {
	ctx, store, client := buildTestableRedisStore[Item](t)
	client.EXPECT().Del(ctx, "foo", "fuu").Return(int64(2), nil)

	items := []string{"foo", "fuu"}

	err := store.DeleteBatch(ctx, items)

	assert.NoError(t, err)
}

func buildTestableRedisStore[T any](t *testing.T) (context.Context, kvstore.KvStore[T], *redisMocks.Client) {
	ctx := t.Context()
	client := redisMocks.NewClient(t)

	store := kvstore.NewRedisKvStoreWithInterfaces[T](client, &kvstore.Settings{
		ModelId: mdl.ModelId{
			Name: "test",
			App:  "app",
			Env:  "env",
			Tags: map[string]string{
				"project": "justtrack",
				"family":  "gosoline",
				"group":   "grp",
			},
		},
		BatchSize:      100,
		MetricsEnabled: false,
	})

	return ctx, store, client
}

func buildTestableRedisStoreWithTTL[T any](t *testing.T) (context.Context, kvstore.KvStore[T], *redisMocks.Client) {
	ctx := t.Context()
	client := redisMocks.NewClient(t)

	store := kvstore.NewRedisKvStoreWithInterfaces[T](client, &kvstore.Settings{
		ModelId: mdl.ModelId{
			Name: "test",
			App:  "kvstore",
			Env:  "test",
			Tags: map[string]string{
				"project": "justtrack",
				"family":  "gosoline",
				"group":   "grp",
			},
		},
		BatchSize:      100,
		MetricsEnabled: false,
		Ttl:            time.Second,
	})

	return ctx, store, client
}
