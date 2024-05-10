package kvstore_test

import (
	"context"
	"testing"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kvstore"
	kvStoreMocks "github.com/justtrackio/gosoline/pkg/kvstore/mocks"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/test/matcher"
	"github.com/stretchr/testify/assert"
)

func TestChainKvStore_Contains(t *testing.T) {
	ctx := t.Context()
	store, element0, element1 := buildTestableChainStore[string](t, false)
	item := new(string)

	// missing
	element0.EXPECT().Get(matcher.Context, "foo", item).Return(false, nil).Once()
	element1.EXPECT().Get(matcher.Context, "foo", item).Return(false, nil).Once()

	exists, err := store.Contains(ctx, "foo")
	assert.NoError(t, err)
	assert.False(t, exists)

	element0.AssertExpectations(t)
	element1.AssertExpectations(t)

	// available
	element0.EXPECT().Get(matcher.Context, "foo", item).Return(false, nil).Once()
	element1.EXPECT().Get(matcher.Context, "foo", item).Run(func(ctx context.Context, key any, value *string) {
		*value = "bar"
	}).Return(true, nil).Once()

	element0.EXPECT().Put(matcher.Context, "foo", "bar").Return(nil).Once()

	exists, err = store.Contains(ctx, "foo")
	assert.NoError(t, err)
	assert.True(t, exists)

	element0.AssertExpectations(t)
	element1.AssertExpectations(t)
}

func TestChainKvStore_Contains_Early(t *testing.T) {
	ctx := t.Context()
	store, element0, element1 := buildTestableChainStore[string](t, false)
	item := new(string)

	element0.EXPECT().Get(matcher.Context, "foo", item).Run(func(ctx context.Context, key any, value *string) {
		*value = "bar"
	}).Return(true, nil).Once()

	exists, err := store.Contains(ctx, "foo")
	assert.NoError(t, err)
	assert.True(t, exists)

	element0.AssertExpectations(t)
	element1.AssertExpectations(t)
}

func TestChainKvStore_Contains_CacheMissing(t *testing.T) {
	ctx := t.Context()
	store, element0, element1 := buildTestableChainStore[string](t, true)
	item := new(string)

	element0.EXPECT().Get(matcher.Context, "foo", item).Return(false, nil).Once()
	element1.EXPECT().Get(matcher.Context, "foo", item).Return(false, nil).Once()

	exists, err := store.Contains(ctx, "foo")
	assert.NoError(t, err)
	assert.False(t, exists)

	element0.AssertExpectations(t)
	element1.AssertExpectations(t)

	exists, err = store.Contains(ctx, "foo")
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestChainKvStore_Get(t *testing.T) {
	ctx := t.Context()
	item := &Item{}
	store, element0, element1 := buildTestableChainStore[Item](t, false)

	// missing
	element0.EXPECT().Get(matcher.Context, "foo", item).Return(false, nil).Once()
	element1.EXPECT().Get(matcher.Context, "foo", item).Return(false, nil).Once()

	found, err := store.Get(ctx, "foo", item)

	assert.NoError(t, err)
	assert.False(t, found)

	// available
	element0.EXPECT().Get(matcher.Context, "foo", item).Return(false, nil).Once()

	element1.EXPECT().Get(matcher.Context, "foo", item).Run(func(ctx context.Context, key any, item *Item) {
		item.Id = "foo"
		item.Body = "bar"
	}).Return(true, nil).Once()

	element0.EXPECT().Put(matcher.Context, "foo", Item{Id: "foo", Body: "bar"}).Return(nil).Once()

	found, err = store.Get(ctx, "foo", item)

	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "foo", item.Id)
	assert.Equal(t, "bar", item.Body)

	element0.AssertExpectations(t)
	element1.AssertExpectations(t)
}

func TestChainKvStore_Get_CacheMissing(t *testing.T) {
	ctx := t.Context()
	item := &Item{}
	store, element0, element1 := buildTestableChainStore[Item](t, true)

	// missing
	element0.EXPECT().Get(matcher.Context, "foo", item).Return(false, nil).Once()
	element1.EXPECT().Get(matcher.Context, "foo", item).Return(false, nil).Once()

	found, err := store.Get(ctx, "foo", item)

	assert.NoError(t, err)
	assert.False(t, found)
	assert.Empty(t, item.Id)

	// available
	element0.EXPECT().Get(matcher.Context, "bar", item).Return(false, nil).Once()

	element1.EXPECT().Get(matcher.Context, "bar", item).Run(func(ctx context.Context, key any, item *Item) {
		item.Id = "bar"
		item.Body = "bar"
	}).Return(true, nil).Once()

	element0.EXPECT().Put(matcher.Context, "bar", Item{Id: "bar", Body: "bar"}).Return(nil).Once()

	found, err = store.Get(ctx, "bar", item)

	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "bar", item.Id)
	assert.Equal(t, "bar", item.Body)

	element0.AssertExpectations(t)
	element1.AssertExpectations(t)
}

func TestChainKvStore_GetBatch(t *testing.T) {
	ctx := t.Context()
	keys := []any{"foo", "fuu"}
	result := make(map[string]Item)

	store, element0, element1 := buildTestableChainStore[Item](t, false)

	// missing all
	element0.EXPECT().GetBatch(matcher.Context, keys, &result).Return(keys, nil).Once()
	element1.EXPECT().GetBatch(matcher.Context, keys, &result).Return(keys, nil).Once()

	missing, err := store.GetBatch(ctx, keys, &result)

	assert.NoError(t, err)
	assert.Len(t, missing, 2)
	assert.Equal(t, keys, missing)

	// missing one
	existing := map[any]Item{
		"foo": {
			Id:   "foo",
			Body: "bar",
		},
	}

	element0.EXPECT().GetBatch(matcher.Context, keys, &result).Return(keys, nil).Once()
	element0.EXPECT().PutBatch(matcher.Context, existing).Return(nil).Once()

	element1.EXPECT().GetBatch(matcher.Context, keys, &result).Run(func(ctx context.Context, keys any, values any) {
		items := values.(*map[string]Item)

		*items = map[string]Item{
			"foo": {
				Id:   "foo",
				Body: "bar",
			},
		}
	}).Return([]any{"fuu"}, nil).Once()

	missing, err = store.GetBatch(ctx, keys, &result)

	assert.NoError(t, err)
	assert.Len(t, missing, 1)
	assert.Contains(t, missing, "fuu")

	assert.Contains(t, result, "foo")
	assert.Equal(t, "foo", result["foo"].Id)
	assert.Equal(t, "bar", result["foo"].Body)

	// missing none
	element0.EXPECT().GetBatch(matcher.Context, keys, &result).Run(func(ctx context.Context, keys any, values any) {
		items := values.(*map[string]Item)

		*items = map[string]Item{
			"foo": {
				Id:   "foo",
				Body: "bar",
			},
			"fuu": {
				Id:   "fuu",
				Body: "baz",
			},
		}
	}).Return([]any{}, nil).Once()

	missing, err = store.GetBatch(ctx, keys, &result)

	assert.NoError(t, err)
	assert.Len(t, missing, 0)

	assert.Contains(t, result, "foo")
	assert.Equal(t, "foo", result["foo"].Id)
	assert.Equal(t, "bar", result["foo"].Body)
	assert.Contains(t, result, "foo")
	assert.Equal(t, "fuu", result["fuu"].Id)
	assert.Equal(t, "baz", result["fuu"].Body)

	element0.AssertExpectations(t)
	element1.AssertExpectations(t)
}

func TestChainKvStore_GetBatch_CacheMissing_All(t *testing.T) {
	expectedMissingKeys := []any{"foo", "fuu"}

	ctx := t.Context()
	keys := []any{"foo", "fuu"}
	result := make(map[string]Item)

	store, element0, element1 := buildTestableChainStore[Item](t, true)

	// missing all
	element0.EXPECT().GetBatch(matcher.Context, keys, result).Return(keys, nil).Once()
	element1.EXPECT().GetBatch(matcher.Context, keys, result).Return(keys, nil).Once()

	missing, err := store.GetBatch(ctx, keys, result)

	assert.NoError(t, err)
	assert.Len(t, missing, 2)
	assert.Equal(t, expectedMissingKeys, missing)

	assert.NotContains(t, result, "foo")
	assert.NotContains(t, result, "fuu")

	element0.AssertExpectations(t)
	element1.AssertExpectations(t)
}

func TestChainKvStore_GetBatch_CacheMissing_None(t *testing.T) {
	ctx := t.Context()
	keys := []any{"foo", "fuu"}
	result := make(map[string]Item)

	store, element0, element1 := buildTestableChainStore[Item](t, true)

	// missing none
	element0.EXPECT().GetBatch(matcher.Context, keys, result).Run(func(ctx context.Context, keys any, values any) {
		items := values.(map[string]Item)

		items["foo"] = Item{
			Id:   "foo",
			Body: "bar",
		}

		items["fuu"] = Item{
			Id:   "fuu",
			Body: "baz",
		}
	}).Return([]any{}, nil).Once()

	missing, err := store.GetBatch(ctx, keys, result)

	assert.NoError(t, err)
	assert.Len(t, missing, 0)

	assert.Contains(t, result, "foo")
	assert.Equal(t, "foo", result["foo"].Id)
	assert.Equal(t, "bar", result["foo"].Body)
	assert.Contains(t, result, "foo")
	assert.Equal(t, "fuu", result["fuu"].Id)
	assert.Equal(t, "baz", result["fuu"].Body)

	element0.AssertExpectations(t)
	element1.AssertExpectations(t)
}

func TestChainKvStore_GetBatch_CacheMissing_One(t *testing.T) {
	expectedMissingKeys := []any{"fuu"}

	ctx := t.Context()
	keys := []any{"foo", "fuu"}
	existingKeys := []any{"foo"}
	result := make(map[string]Item)

	store, element0, element1 := buildTestableChainStore[Item](t, true)

	// missing one
	existing := map[any]Item{
		"foo": {
			Id:   "foo",
			Body: "bar",
		},
	}

	element0.EXPECT().GetBatch(matcher.Context, keys, &result).Return(keys, nil).Once()

	element1.EXPECT().GetBatch(matcher.Context, keys, &result).Run(func(ctx context.Context, keys any, values any) {
		items := values.(*map[string]Item)

		*items = map[string]Item{
			"foo": {
				Id:   "foo",
				Body: "bar",
			},
		}
	}).Return([]any{"fuu"}, nil).Once()

	element0.EXPECT().PutBatch(matcher.Context, existing).Return(nil).Once()

	missing, err := store.GetBatch(ctx, keys, &result)

	assert.NoError(t, err)
	assert.Len(t, missing, 1)
	assert.Equal(t, expectedMissingKeys, missing)

	assert.Contains(t, result, "foo")
	assert.NotContains(t, result, "fuu")
	assert.Equal(t, "foo", result["foo"].Id)
	assert.Equal(t, "bar", result["foo"].Body)

	element0.AssertExpectations(t)
	element1.AssertExpectations(t)

	element0.EXPECT().GetBatch(matcher.Context, existingKeys, &result).Run(func(ctx context.Context, keys any, values any) {
		items := values.(*map[string]Item)

		*items = map[string]Item{
			"foo": {
				Id:   "foo",
				Body: "bar",
			},
		}
	}).Return(nil, nil).Once()

	missing, err = store.GetBatch(ctx, keys, &result)

	assert.NoError(t, err)
	assert.Len(t, missing, 1)
	assert.Equal(t, expectedMissingKeys, missing)

	assert.Contains(t, result, "foo")
	assert.NotContains(t, result, "fuu")
	assert.Equal(t, "foo", result["foo"].Id)
	assert.Equal(t, "bar", result["foo"].Body)

	element0.AssertExpectations(t)
}

func TestChainKvStore_Put(t *testing.T) {
	ctx := t.Context()
	item := Item{
		Id:   "foo",
		Body: "bar",
	}

	store, element0, element1 := buildTestableChainStore[Item](t, false)

	element0.EXPECT().Put(matcher.Context, "foo", item).Return(nil).Once()
	element1.EXPECT().Put(matcher.Context, "foo", item).Return(nil).Once()

	err := store.Put(ctx, "foo", item)

	assert.NoError(t, err)
	element0.AssertExpectations(t)
	element1.AssertExpectations(t)
}

func TestChainKvStore_PutBatch(t *testing.T) {
	ctx := t.Context()
	items := map[string]Item{
		"fuu": {
			Id:   "fuu",
			Body: "baz",
		},
		"foo": {
			Id:   "foo",
			Body: "bar",
		},
	}
	itemsII := map[any]any{}
	for k, v := range items {
		itemsII[k] = v
	}

	store, element0, element1 := buildTestableChainStore[Item](t, false)

	element0.EXPECT().PutBatch(matcher.Context, itemsII).Return(nil).Once()
	element1.EXPECT().PutBatch(matcher.Context, itemsII).Return(nil).Once()

	err := store.PutBatch(ctx, items)

	assert.NoError(t, err)
	element0.AssertExpectations(t)
	element1.AssertExpectations(t)
}

func TestChainKvStore_Delete(t *testing.T) {
	ctx := t.Context()

	store, element0, element1 := buildTestableChainStore[string](t, false)

	element0.EXPECT().Delete(matcher.Context, "foo").Return(nil).Once()
	element1.EXPECT().Delete(matcher.Context, "foo").Return(nil).Once()

	err := store.Delete(ctx, "foo")

	assert.NoError(t, err)
	element0.AssertExpectations(t)
	element1.AssertExpectations(t)
}

func TestChainKvStore_DeleteBatch(t *testing.T) {
	ctx := t.Context()
	items := []string{"fuu", "foo"}

	store, element0, element1 := buildTestableChainStore[string](t, false)

	element0.EXPECT().DeleteBatch(matcher.Context, items).Return(nil).Once()
	element1.EXPECT().DeleteBatch(matcher.Context, items).Return(nil).Once()

	err := store.DeleteBatch(ctx, items)

	assert.NoError(t, err)
	element0.AssertExpectations(t)
	element1.AssertExpectations(t)
}

func nilFactory[T any](_ kvstore.ElementFactory[T], _ *kvstore.Settings) (kvstore.KvStore[T], error) {
	return nil, nil
}

func buildTestableChainStore[T any](t *testing.T, missingCacheEnabled bool) (
	store kvstore.ChainKvStore[T],
	element0 *kvStoreMocks.KvStore[T],
	element1 *kvStoreMocks.KvStore[T],
) {
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll)

	element0 = kvStoreMocks.NewKvStore[T](t)
	element1 = kvStoreMocks.NewKvStore[T](t)

	settings := &kvstore.Settings{
		AppId: cfg.AppId{
			Project:     "applike",
			Environment: "test",
			Family:      "gosoline",
			Group:       "grp",
			Application: "kvstore",
		},
		Name:      "test",
		BatchSize: 100,
	}

	var missingCache kvstore.KvStore[T]
	if missingCacheEnabled {
		missingCache = kvstore.NewInMemoryKvStoreWithInterfaces[T](settings)
	} else {
		missingCache = kvstore.NewEmptyKvStore[T]()
	}

	store = kvstore.NewChainKvStoreWithInterfaces[T](logger, nilFactory[T], missingCache, settings)

	store.AddStore(element0)
	store.AddStore(element1)

	return store, element0, element1
}
