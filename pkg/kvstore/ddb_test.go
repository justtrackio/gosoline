package kvstore_test

import (
	"context"
	"testing"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/ddb"
	ddbMocks "github.com/justtrackio/gosoline/pkg/ddb/mocks"
	"github.com/justtrackio/gosoline/pkg/kvstore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestDdbKvStore_Contains(t *testing.T) {
	store, repo := buildTestableDdbStore[string]()

	builder := new(ddbMocks.GetItemBuilder)
	builder.On("WithHash", "foo").Return(builder).Once()

	repo.On("GetItemBuilder").Return(builder)
	repo.On("GetItem", mock.AnythingOfType("*context.emptyCtx"), builder, mock.AnythingOfType("*kvstore.DdbItem")).Return(&ddb.GetItemResult{
		IsFound: true,
	}, nil).Once()

	exists, err := store.Contains(context.Background(), "foo")
	assert.NoError(t, err)
	assert.True(t, exists)

	builder.On("WithHash", "fuu").Return(builder).Once()
	repo.On("GetItem", mock.AnythingOfType("*context.emptyCtx"), builder, mock.AnythingOfType("*kvstore.DdbItem")).Return(&ddb.GetItemResult{
		IsFound: false,
	}, nil).Once()

	exists, err = store.Contains(context.Background(), "fuu")
	assert.NoError(t, err)
	assert.False(t, exists)

	builder.AssertExpectations(t)
	repo.AssertExpectations(t)
}

func TestDdbKvStore_Get(t *testing.T) {
	store, repo := buildTestableDdbStore[Item]()

	builder := new(ddbMocks.GetItemBuilder)
	builder.On("WithHash", "foo").Return(builder).Once()

	ddbItem := &kvstore.DdbItem{
		Key:   "",
		Value: "",
	}

	repo.On("GetItemBuilder").Return(builder)
	repo.On("GetItem", mock.AnythingOfType("*context.emptyCtx"), builder, ddbItem).Run(func(args mock.Arguments) {
		ddbItem := args[2].(*kvstore.DdbItem)
		ddbItem.Key = "foo"
		ddbItem.Value = `{"id":"foo","body":"bar"}`
	}).Return(&ddb.GetItemResult{
		IsFound: true,
	}, nil).Once()

	item := &Item{}
	found, err := store.Get(context.Background(), "foo", item)

	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "foo", item.Id)
	assert.Equal(t, "bar", item.Body)

	builder.AssertExpectations(t)
	repo.AssertExpectations(t)
}

func TestDdbKvStore_GetBatch(t *testing.T) {
	store, repo := buildTestableDdbStore[Item]()

	keys := []string{"foo", "fuu"}
	result := make(map[string]Item)

	builder := new(ddbMocks.BatchGetItemsBuilder)
	builder.On("WithHashKeys", keys).Return(builder)

	items := make([]kvstore.DdbItem, 0)

	repo.On("BatchGetItemsBuilder").Return(builder)
	repo.On("BatchGetItems", mock.AnythingOfType("*context.emptyCtx"), builder, &items).Run(func(args mock.Arguments) {
		item := kvstore.DdbItem{
			Key:   "foo",
			Value: `{"id":"foo","body":"bar"}`,
		}
		items := args[2].(*[]kvstore.DdbItem)
		*items = append(*items, item)
	}).Return(nil, nil)

	missing, err := store.GetBatch(context.Background(), keys, result)

	assert.NoError(t, err)
	assert.Contains(t, result, "foo")
	assert.Equal(t, "foo", result["foo"].Id)
	assert.Equal(t, "bar", result["foo"].Body)

	assert.Len(t, missing, 1)
	assert.Contains(t, missing, "fuu")

	builder.AssertExpectations(t)
	repo.AssertExpectations(t)
}

func TestDdbKvStore_GetBatch_ReturnedKeysInDifferentOrder(t *testing.T) {
	store, repo := buildTestableDdbStore[Item]()

	result := make(map[string]*Item)

	builder := new(ddbMocks.BatchGetItemsBuilder)
	builder.On("WithHashKeys", []string{"foo", "fuu"}).Return(builder)

	items := make([]kvstore.DdbItem, 0)

	repo.On("BatchGetItemsBuilder").Return(builder)

	// the order of the entries is not always the same as the order of the keys
	repo.On("BatchGetItems", mock.AnythingOfType("*context.emptyCtx"), builder, &items).Run(func(args mock.Arguments) {
		items := args[2].(*[]kvstore.DdbItem)
		*items = append(*items, kvstore.DdbItem{
			Key:   "fuu",
			Value: `{"id":"fuu","body":"baz"}`,
		})
		*items = append(*items, kvstore.DdbItem{
			Key:   "foo",
			Value: `{"id":"foo","body":"bar"}`,
		})
	}).Return(nil, nil)

	missing, err := store.GetBatch(context.Background(), []string{"foo", "fuu", "foo"}, result)

	assert.NoError(t, err)

	assert.Contains(t, result, "foo")
	assert.Equal(t, "foo", result["foo"].Id)
	assert.Equal(t, "bar", result["foo"].Body)

	assert.Contains(t, result, "fuu")
	assert.Equal(t, "fuu", result["fuu"].Id)
	assert.Equal(t, "baz", result["fuu"].Body)

	assert.Len(t, missing, 0)

	builder.AssertExpectations(t)
	repo.AssertExpectations(t)
}

func TestDdbKvStore_GetBatch_WithDuplicateKeys(t *testing.T) {
	store, repo := buildTestableDdbStore[Item]()

	result := make(map[string]*Item)

	builder := new(ddbMocks.BatchGetItemsBuilder)
	builder.On("WithHashKeys", []string{"foo", "fuu"}).Return(builder)

	items := make([]kvstore.DdbItem, 0)

	repo.On("BatchGetItemsBuilder").Return(builder)
	repo.On("BatchGetItems", mock.AnythingOfType("*context.emptyCtx"), builder, &items).Run(func(args mock.Arguments) {
		item := kvstore.DdbItem{
			Key:   "foo",
			Value: `{"id":"foo","body":"bar"}`,
		}

		items := args[2].(*[]kvstore.DdbItem)
		*items = append(*items, item)
	}).Return(nil, nil)

	missing, err := store.GetBatch(context.Background(), []string{"foo", "fuu", "foo"}, result)

	assert.NoError(t, err)
	assert.Contains(t, result, "foo")
	assert.Equal(t, "foo", result["foo"].Id)
	assert.Equal(t, "bar", result["foo"].Body)

	assert.Len(t, missing, 1)
	assert.Contains(t, missing, "fuu")

	builder.AssertExpectations(t)
	repo.AssertExpectations(t)
}

func TestDdbKvStore_Put(t *testing.T) {
	store, repo := buildTestableDdbStore[Item]()

	ddbItem := &kvstore.DdbItem{
		Key:   "foo",
		Value: `{"id":"foo","body":"bar"}`,
	}
	repo.On("PutItem", mock.AnythingOfType("*context.emptyCtx"), nil, ddbItem).Return(nil, nil)

	item := Item{
		Id:   "foo",
		Body: "bar",
	}

	err := store.Put(context.Background(), "foo", item)

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestDdbKvStore_PutBatch(t *testing.T) {
	store, repo := buildTestableDdbStore[Item]()

	ddbItems := []kvstore.DdbItem{
		{
			Key:   "foo",
			Value: `{"id":"foo","body":"bar"}`,
		},
		{
			Key:   "fuu",
			Value: `{"id":"fuu","body":"baz"}`,
		},
	}
	repo.On("BatchPutItems", mock.AnythingOfType("*context.emptyCtx"), ddbItems).Return(nil, nil)

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

	err := store.PutBatch(context.Background(), items)

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestDdbKvStore_Delete(t *testing.T) {
	store, repo := buildTestableDdbStore[Item]()

	ddbItem := &kvstore.DdbDeleteItem{
		Key: "foo",
	}
	repo.On("DeleteItem", mock.AnythingOfType("*context.emptyCtx"), nil, ddbItem).Return(nil, nil)

	err := store.Delete(context.Background(), "foo")

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestDdbKvStore_DeleteBatch(t *testing.T) {
	store, repo := buildTestableDdbStore[string]()

	ddbItems := []*kvstore.DdbDeleteItem{
		{
			Key: "foo",
		},
		{
			Key: "fuu",
		},
	}
	repo.On("BatchDeleteItems", mock.AnythingOfType("*context.emptyCtx"), ddbItems).Return(nil, nil)

	items := []string{"foo", "fuu"}

	err := store.DeleteBatch(context.Background(), items)

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func buildTestableDdbStore[T any]() (kvstore.KvStore[T], *ddbMocks.Repository) {
	repository := new(ddbMocks.Repository)

	store := kvstore.NewDdbKvStoreWithInterfaces[T](repository, &kvstore.Settings{
		AppId: cfg.AppId{
			Project:     "applike",
			Environment: "test",
			Family:      "gosoline",
			Group:       "grp",
			Application: "kvstore",
		},
		Name:      "test",
		BatchSize: 100,
	})

	return store, repository
}
