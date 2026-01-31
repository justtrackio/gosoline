package kvstore_test

import (
	"context"
	"testing"

	"github.com/justtrackio/gosoline/pkg/ddb"
	ddbMocks "github.com/justtrackio/gosoline/pkg/ddb/mocks"
	"github.com/justtrackio/gosoline/pkg/kvstore"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestDdbKvStore_Contains(t *testing.T) {
	ctx, store, repo := buildTestableDdbStore[string](t)

	builder := ddbMocks.NewGetItemBuilder(t)
	builder.EXPECT().WithHash("foo").Return(builder).Once()

	repo.EXPECT().GetItemBuilder().Return(builder)
	repo.EXPECT().GetItem(ctx, builder, mock.AnythingOfType("*kvstore.DdbItem")).Return(&ddb.GetItemResult{
		IsFound: true,
	}, nil).Once()

	exists, err := store.Contains(ctx, "foo")
	assert.NoError(t, err)
	assert.True(t, exists)

	builder.EXPECT().WithHash("fuu").Return(builder).Once()
	repo.EXPECT().GetItem(ctx, builder, mock.AnythingOfType("*kvstore.DdbItem")).Return(&ddb.GetItemResult{
		IsFound: false,
	}, nil).Once()

	exists, err = store.Contains(ctx, "fuu")
	assert.NoError(t, err)
	assert.False(t, exists)

	builder.AssertExpectations(t)
	repo.AssertExpectations(t)
}

func TestDdbKvStore_Get(t *testing.T) {
	ctx, store, repo := buildTestableDdbStore[Item](t)

	builder := ddbMocks.NewGetItemBuilder(t)
	builder.EXPECT().WithHash("foo").Return(builder).Once()

	ddbItem := &kvstore.DdbItem{
		Key:   "",
		Value: "",
	}

	repo.EXPECT().GetItemBuilder().Return(builder)
	repo.EXPECT().GetItem(ctx, builder, ddbItem).Run(func(ctx context.Context, qb ddb.GetItemBuilder, result any) {
		ddbItem := result.(*kvstore.DdbItem)
		ddbItem.Key = "foo"
		ddbItem.Value = `{"id":"foo","body":"bar"}`
	}).Return(&ddb.GetItemResult{
		IsFound: true,
	}, nil).Once()

	item := &Item{}
	found, err := store.Get(ctx, "foo", item)

	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "foo", item.Id)
	assert.Equal(t, "bar", item.Body)

	builder.AssertExpectations(t)
	repo.AssertExpectations(t)
}

func TestDdbKvStore_GetBatch(t *testing.T) {
	ctx, store, repo := buildTestableDdbStore[Item](t)

	keys := []string{"foo", "fuu"}
	result := make(map[string]Item)

	builder := ddbMocks.NewBatchGetItemsBuilder(t)
	builder.EXPECT().WithHashKeys(keys).Return(builder)

	items := make([]kvstore.DdbItem, 0)

	repo.EXPECT().BatchGetItemsBuilder().Return(builder)
	repo.EXPECT().BatchGetItems(ctx, builder, &items).Run(func(ctx context.Context, qb ddb.BatchGetItemsBuilder, result any) {
		item := kvstore.DdbItem{
			Key:   "foo",
			Value: `{"id":"foo","body":"bar"}`,
		}
		items := result.(*[]kvstore.DdbItem)
		*items = append(*items, item)
	}).Return(nil, nil)

	missing, err := store.GetBatch(ctx, keys, result)

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
	ctx, store, repo := buildTestableDdbStore[Item](t)

	result := make(map[string]*Item)

	builder := ddbMocks.NewBatchGetItemsBuilder(t)
	builder.EXPECT().WithHashKeys([]string{"foo", "fuu"}).Return(builder)

	items := make([]kvstore.DdbItem, 0)

	repo.EXPECT().BatchGetItemsBuilder().Return(builder)

	// the order of the entries is not always the same as the order of the keys
	repo.EXPECT().BatchGetItems(ctx, builder, &items).Run(func(ctx context.Context, qb ddb.BatchGetItemsBuilder, result any) {
		items := result.(*[]kvstore.DdbItem)
		*items = append(*items, kvstore.DdbItem{
			Key:   "fuu",
			Value: `{"id":"fuu","body":"baz"}`,
		})
		*items = append(*items, kvstore.DdbItem{
			Key:   "foo",
			Value: `{"id":"foo","body":"bar"}`,
		})
	}).Return(nil, nil)

	missing, err := store.GetBatch(ctx, []string{"foo", "fuu", "foo"}, result)

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
	ctx, store, repo := buildTestableDdbStore[Item](t)

	result := make(map[string]*Item)

	builder := ddbMocks.NewBatchGetItemsBuilder(t)
	builder.EXPECT().WithHashKeys([]string{"foo", "fuu"}).Return(builder)

	items := make([]kvstore.DdbItem, 0)

	repo.EXPECT().BatchGetItemsBuilder().Return(builder)
	repo.EXPECT().BatchGetItems(ctx, builder, &items).Run(func(ctx context.Context, qb ddb.BatchGetItemsBuilder, result any) {
		item := kvstore.DdbItem{
			Key:   "foo",
			Value: `{"id":"foo","body":"bar"}`,
		}

		items := result.(*[]kvstore.DdbItem)
		*items = append(*items, item)
	}).Return(nil, nil)

	missing, err := store.GetBatch(ctx, []string{"foo", "fuu", "foo"}, result)

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
	ctx, store, repo := buildTestableDdbStore[Item](t)

	ddbItem := &kvstore.DdbItem{
		Key:   "foo",
		Value: `{"id":"foo","body":"bar"}`,
	}
	repo.EXPECT().PutItem(ctx, nil, ddbItem).Return(nil, nil)

	item := Item{
		Id:   "foo",
		Body: "bar",
	}

	err := store.Put(ctx, "foo", item)

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestDdbKvStore_PutBatch(t *testing.T) {
	ctx, store, repo := buildTestableDdbStore[Item](t)

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
	repo.EXPECT().BatchPutItems(ctx, ddbItems).Return(nil, nil)

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

	err := store.PutBatch(ctx, items)

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestDdbKvStore_Delete(t *testing.T) {
	ctx, store, repo := buildTestableDdbStore[Item](t)

	ddbItem := &kvstore.DdbDeleteItem{
		Key: "foo",
	}
	repo.EXPECT().DeleteItem(ctx, nil, ddbItem).Return(nil, nil)

	err := store.Delete(ctx, "foo")

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestDdbKvStore_DeleteBatch(t *testing.T) {
	ctx, store, repo := buildTestableDdbStore[string](t)

	ddbItems := []*kvstore.DdbDeleteItem{
		{
			Key: "foo",
		},
		{
			Key: "fuu",
		},
	}
	repo.EXPECT().BatchDeleteItems(ctx, ddbItems).Return(nil, nil)

	items := []string{"foo", "fuu"}

	err := store.DeleteBatch(ctx, items)

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func buildTestableDdbStore[T any](t *testing.T) (context.Context, kvstore.KvStore[T], *ddbMocks.Repository) {
	ctx := t.Context()
	repository := ddbMocks.NewRepository(t)

	store := kvstore.NewDdbKvStoreWithInterfaces[T](repository, &kvstore.Settings{
		ModelId: mdl.ModelId{
			Name: "test",
			App:  "kvstore",
			Tags: map[string]string{
				"project": "applike",
				"family":  "gosoline",
				"group":   "grp",
			},
		},
		BatchSize: 100,
	})

	return ctx, store, repository
}
