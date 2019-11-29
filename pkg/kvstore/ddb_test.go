package kvstore_test

import (
	"context"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/ddb"
	ddbMocks "github.com/applike/gosoline/pkg/ddb/mocks"
	"github.com/applike/gosoline/pkg/kvstore"
	monMocks "github.com/applike/gosoline/pkg/mon/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

func TestDdbKvStore_Contains(t *testing.T) {
	store, repo := buildTestableDdbStore()

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
	store, repo := buildTestableDdbStore()

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
	store, repo := buildTestableDdbStore()

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

func TestDdbKvStore_Put(t *testing.T) {
	store, repo := buildTestableDdbStore()

	ddbItem := &kvstore.DdbItem{
		Key:   "foo",
		Value: `{"id":"foo","body":"bar"}`,
	}
	repo.On("PutItem", mock.AnythingOfType("*context.emptyCtx"), nil, ddbItem).Return(nil, nil)

	item := &Item{
		Id:   "foo",
		Body: "bar",
	}

	err := store.Put(context.Background(), "foo", item)

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestDdbKvStore_PutBatch(t *testing.T) {
	store, repo := buildTestableDdbStore()

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

func buildTestableDdbStore() (*kvstore.DdbKvStore, *ddbMocks.Repository) {
	logger := monMocks.NewLoggerMockedAll()
	repository := new(ddbMocks.Repository)

	store := kvstore.NewDdbKvStoreWithInterfaces(logger, repository, &kvstore.Settings{
		AppId: cfg.AppId{
			Project:     "applike",
			Environment: "test",
			Family:      "gosoline",
			Application: "kvstore",
		},
		Name:      "test",
		BatchSize: 100,
	})

	return store, repository
}
