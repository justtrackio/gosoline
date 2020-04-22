package kvstore_test

import (
	"context"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/kvstore"
	"github.com/stretchr/testify/suite"
	"testing"
	"time"
)

type InMemoryKvStoreTestSuite struct {
	suite.Suite
	store *kvstore.InMemoryKvStore
}

func (s *InMemoryKvStoreTestSuite) SetupTest() {
	s.store = kvstore.NewInMemoryKvStoreWithInterfaces(&kvstore.Settings{
		AppId:     cfg.AppId{},
		Name:      "",
		Ttl:       time.Hour,
		BatchSize: 100,
	})
}

func (s *InMemoryKvStoreTestSuite) TestStoreBasic() {
	ctx := context.Background()

	err := s.store.Put(ctx, "key", 1.1)
	s.NoError(err, "there should be no error on Put")

	var v1 float64
	ok, err := s.store.Get(ctx, "key", &v1)
	s.NoError(err, "there should be no error on Get")
	s.True(ok, "the item should be in the store")

	var v2 = 1.2
	err = s.store.Put(ctx, "key", &v2)
	s.NoError(err, "there should be no error on Put")

	var v3 float64
	ok, err = s.store.Get(ctx, "key", &v3)
	s.NoError(err, "there should be no error on Get")
	s.True(ok, "the item should be in the store")
}

func (s *InMemoryKvStoreTestSuite) TestStoreStruct() {
	ctx := context.Background()

	type item struct {
		i int
		s string
	}

	a := item{i: 1, s: "s"}
	err := s.store.Put(ctx, "key", a)
	s.NoError(err, "there should be no error on Put")

	ok, err := s.store.Contains(ctx, "key")
	s.NoError(err, "there should be no error on Contains")
	s.True(ok, "the item should be in the store")

	ok, err = s.store.Contains(ctx, "missing")
	s.NoError(err, "there should be no error on Contains")
	s.False(ok, "the item should be missing the store")

	aCpy := item{}
	ok, err = s.store.Get(ctx, "key", &aCpy)
	s.NoError(err, "there should be no error on Get")
	s.True(ok, "the item should be in the store")
	s.Equal(a, aCpy, "the retrieved item should match the original one")

	putBatch := map[string]interface{}{
		"b": item{i: 2, s: "t"},
		"c": item{i: 3, s: "u"},
	}
	err = s.store.PutBatch(ctx, putBatch)
	s.NoError(err, "there should be no error on PutBatch")

	getBatch := make(map[string]item)
	missing, err := s.store.GetBatch(ctx, []string{"b", "c", "d"}, getBatch)
	s.NoError(err, "there should be no error on GetBatch")
	s.Len(getBatch, 2, "there should be 2 elements in the batch")
	s.Equal(putBatch["b"], getBatch["b"])
	s.Equal(putBatch["c"], getBatch["c"])
	s.Len(missing, 1, "there should be 1 missing element")
	s.Equal("d", missing[0], "element d should be missing")
}

func TestInMemoryKvStoreTestSuite(t *testing.T) {
	suite.Run(t, new(InMemoryKvStoreTestSuite))
}
