package kvstore_test

import (
	"context"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/kvstore"
	kvStoreMocks "github.com/applike/gosoline/pkg/kvstore/mocks"
	logMocks "github.com/applike/gosoline/pkg/log/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

func TestChainKvStore_Contains(t *testing.T) {
	ctx := context.Background()
	store, element0, element1 := buildTestableChainStore(false)

	// missing
	element0.On("Contains", ctx, "foo").Return(false, nil).Once()
	element1.On("Contains", ctx, "foo").Return(false, nil).Once()

	exists, err := store.Contains(ctx, "foo")
	assert.NoError(t, err)
	assert.False(t, exists)

	element0.AssertExpectations(t)
	element1.AssertExpectations(t)

	// available
	element0.On("Contains", ctx, "foo").Return(false, nil).Once()
	element1.On("Contains", ctx, "foo").Return(true, nil).Once()

	exists, err = store.Contains(ctx, "foo")
	assert.NoError(t, err)
	assert.True(t, exists)

	element0.AssertExpectations(t)
	element1.AssertExpectations(t)
}

func TestChainKvStore_Contains_Early(t *testing.T) {
	ctx := context.Background()
	store, element0, element1 := buildTestableChainStore(false)

	element0.On("Contains", ctx, "foo").Return(true, nil).Once()

	exists, err := store.Contains(ctx, "foo")
	assert.NoError(t, err)
	assert.True(t, exists)

	element0.AssertExpectations(t)
	element1.AssertExpectations(t)
}

func TestChainKvStore_Contains_CacheMissing(t *testing.T) {
	ctx := context.Background()
	store, element0, element1 := buildTestableChainStore(true)

	element0.On("Contains", ctx, "foo").Return(false, nil).Once()
	element1.On("Contains", ctx, "foo").Return(false, nil).Once()

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
	ctx := context.Background()
	item := &Item{}
	store, element0, element1 := buildTestableChainStore(false)

	// missing
	element0.On("Get", ctx, "foo", item).Return(false, nil).Once()
	element1.On("Get", ctx, "foo", item).Return(false, nil).Once()

	found, err := store.Get(ctx, "foo", item)

	assert.NoError(t, err)
	assert.False(t, found)

	// available
	element0.On("Get", ctx, "foo", item).Return(false, nil).Once()
	element0.On("Put", ctx, "foo", item).Return(nil)

	element1.On("Get", ctx, "foo", item).Run(func(args mock.Arguments) {
		item := args[2].(*Item)
		item.Id = "foo"
		item.Body = "bar"
	}).Return(true, nil).Once()

	found, err = store.Get(ctx, "foo", item)

	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "foo", item.Id)
	assert.Equal(t, "bar", item.Body)

	element0.AssertExpectations(t)
	element1.AssertExpectations(t)
}

func TestChainKvStore_Get_CacheMissing(t *testing.T) {
	ctx := context.Background()
	item := &Item{}
	store, element0, element1 := buildTestableChainStore(true)

	// missing
	element0.On("Get", ctx, "foo", item).Return(false, nil).Once()
	element1.On("Get", ctx, "foo", item).Return(false, nil).Once()

	found, err := store.Get(ctx, "foo", item)

	assert.NoError(t, err)
	assert.False(t, found)
	assert.Empty(t, item.Id)

	// available
	element0.On("Get", ctx, "bar", item).Return(false, nil).Once()
	element0.On("Put", ctx, "bar", item).Return(nil)

	element1.On("Get", ctx, "bar", item).Run(func(args mock.Arguments) {
		item := args[2].(*Item)
		item.Id = "bar"
		item.Body = "bar"
	}).Return(true, nil).Once()

	found, err = store.Get(ctx, "bar", item)

	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "bar", item.Id)
	assert.Equal(t, "bar", item.Body)

	element0.AssertExpectations(t)
	element1.AssertExpectations(t)
}

func TestChainKvStore_GetBatch(t *testing.T) {
	ctx := context.Background()
	keys := []interface{}{"foo", "fuu"}
	result := make(map[string]Item)

	store, element0, element1 := buildTestableChainStore(false)

	// missing all
	element0.On("GetBatch", ctx, keys, result).Return(keys, nil).Once()
	element1.On("GetBatch", ctx, keys, result).Return(keys, nil).Once()

	missing, err := store.GetBatch(ctx, keys, result)

	assert.NoError(t, err)
	assert.Len(t, missing, 2)
	assert.Equal(t, keys, missing)

	// missing one
	existing := map[interface{}]interface{}{
		"foo": Item{
			Id:   "foo",
			Body: "bar",
		},
	}

	element0.On("GetBatch", ctx, keys, result).Return(keys, nil).Once()
	element0.On("PutBatch", ctx, existing).Return(nil).Once()

	element1.On("GetBatch", ctx, keys, result).Run(func(args mock.Arguments) {
		items := args[2].(map[string]Item)

		items["foo"] = Item{
			Id:   "foo",
			Body: "bar",
		}
	}).Return([]interface{}{"fuu"}, nil).Once()

	missing, err = store.GetBatch(ctx, keys, result)

	assert.NoError(t, err)
	assert.Len(t, missing, 1)
	assert.Contains(t, missing, "fuu")

	assert.Contains(t, result, "foo")
	assert.Equal(t, "foo", result["foo"].Id)
	assert.Equal(t, "bar", result["foo"].Body)

	// missing none
	element0.On("GetBatch", ctx, keys, result).Run(func(args mock.Arguments) {
		items := args[2].(map[string]Item)

		items["foo"] = Item{
			Id:   "foo",
			Body: "bar",
		}

		items["fuu"] = Item{
			Id:   "fuu",
			Body: "baz",
		}
	}).Return([]interface{}{}, nil).Once()

	missing, err = store.GetBatch(ctx, keys, result)

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
	expectedMissingKeys := []interface{}{"foo", "fuu"}

	ctx := context.Background()
	keys := []interface{}{"foo", "fuu"}
	result := make(map[string]Item)

	store, element0, element1 := buildTestableChainStore(true)

	// missing all
	element0.On("GetBatch", ctx, keys, result).Return(keys, nil).Once()
	element1.On("GetBatch", ctx, keys, result).Return(keys, nil).Once()

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
	ctx := context.Background()
	keys := []interface{}{"foo", "fuu"}
	result := make(map[string]Item)

	store, element0, element1 := buildTestableChainStore(true)

	// missing none
	element0.On("GetBatch", ctx, keys, result).Run(func(args mock.Arguments) {
		items := args[2].(map[string]Item)

		items["foo"] = Item{
			Id:   "foo",
			Body: "bar",
		}

		items["fuu"] = Item{
			Id:   "fuu",
			Body: "baz",
		}
	}).Return([]interface{}{}, nil).Once()

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
	expectedMissingKeys := []interface{}{"fuu"}

	ctx := context.Background()
	keys := []interface{}{"foo", "fuu"}
	existingKeys := []interface{}{"foo"}
	result := make(map[string]Item)

	store, element0, element1 := buildTestableChainStore(true)

	// missing one
	existing := map[interface{}]interface{}{
		"foo": Item{
			Id:   "foo",
			Body: "bar",
		},
	}

	element0.On("GetBatch", ctx, keys, result).Return(keys, nil).Once()
	element0.On("PutBatch", ctx, existing).Return(nil).Once()

	element1.On("GetBatch", ctx, keys, result).Run(func(args mock.Arguments) {
		items := args[2].(map[string]Item)

		items["foo"] = Item{
			Id:   "foo",
			Body: "bar",
		}
	}).Return([]interface{}{"fuu"}, nil).Once()

	missing, err := store.GetBatch(ctx, keys, result)

	assert.NoError(t, err)
	assert.Len(t, missing, 1)
	assert.Equal(t, expectedMissingKeys, missing)

	assert.Contains(t, result, "foo")
	assert.NotContains(t, result, "fuu")
	assert.Equal(t, "foo", result["foo"].Id)
	assert.Equal(t, "bar", result["foo"].Body)

	element0.AssertExpectations(t)
	element1.AssertExpectations(t)

	element0.On("GetBatch", ctx, existingKeys, result).Run(func(args mock.Arguments) {
		items := args[2].(map[string]Item)

		items["foo"] = Item{
			Id:   "foo",
			Body: "bar",
		}
	}).Return(nil, nil).Once()

	missing, err = store.GetBatch(ctx, keys, result)

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
	ctx := context.Background()
	item := Item{
		Id:   "foo",
		Body: "bar",
	}

	store, element0, element1 := buildTestableChainStore(false)

	element0.On("Put", ctx, "foo", item).Return(nil).Once()
	element1.On("Put", ctx, "foo", item).Return(nil).Once()

	err := store.Put(ctx, "foo", item)

	assert.NoError(t, err)
	element0.AssertExpectations(t)
	element1.AssertExpectations(t)
}

func TestChainKvStore_PutBatch(t *testing.T) {
	ctx := context.Background()
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
	itemsII := map[interface{}]interface{}{}
	for k, v := range items {
		itemsII[k] = v
	}

	store, element0, element1 := buildTestableChainStore(false)

	element0.On("PutBatch", ctx, itemsII).Return(nil).Once()
	element1.On("PutBatch", ctx, itemsII).Return(nil).Once()

	err := store.PutBatch(ctx, items)

	assert.NoError(t, err)
	element0.AssertExpectations(t)
	element1.AssertExpectations(t)
}

func TestChainKvStore_Delete(t *testing.T) {
	ctx := context.Background()

	store, element0, element1 := buildTestableChainStore(false)

	element0.On("Delete", ctx, "foo").Return(nil).Once()
	element1.On("Delete", ctx, "foo").Return(nil).Once()

	err := store.Delete(ctx, "foo")

	assert.NoError(t, err)
	element0.AssertExpectations(t)
	element1.AssertExpectations(t)
}

func TestChainKvStore_DeleteBatch(t *testing.T) {
	ctx := context.Background()
	items := []string{"fuu", "foo"}

	store, element0, element1 := buildTestableChainStore(false)

	element0.On("DeleteBatch", ctx, items).Return(nil).Once()
	element1.On("DeleteBatch", ctx, items).Return(nil).Once()

	err := store.DeleteBatch(ctx, items)

	assert.NoError(t, err)
	element0.AssertExpectations(t)
	element1.AssertExpectations(t)
}

func nilFactory(_ kvstore.Factory, _ *kvstore.Settings) (kvstore.KvStore, error) {
	return nil, nil
}

func buildTestableChainStore(missingCacheEnabled bool) (kvstore.KvStore, *kvStoreMocks.KvStore, *kvStoreMocks.KvStore) {
	logger := logMocks.NewLoggerMockedAll()

	element0 := new(kvStoreMocks.KvStore)
	element1 := new(kvStoreMocks.KvStore)

	settings := &kvstore.Settings{
		AppId: cfg.AppId{
			Project:     "applike",
			Environment: "test",
			Family:      "gosoline",
			Application: "kvstore",
		},
		Name:      "test",
		BatchSize: 100,
	}

	var missingCache kvstore.KvStore
	if missingCacheEnabled {
		missingCache = kvstore.NewInMemoryKvStoreWithInterfaces(settings)
	} else {
		missingCache = kvstore.NewEmptyKvStore()
	}

	store := kvstore.NewChainKvStoreWithInterfaces(logger, nilFactory, missingCache, settings)

	store.AddStore(element0)
	store.AddStore(element1)

	return store, element0, element1
}
