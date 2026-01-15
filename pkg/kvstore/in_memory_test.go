package kvstore_test

import (
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/kvstore"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/stretchr/testify/suite"
)

type item struct {
	i int
	s string
}

type InMemoryKvStoreTestSuite struct {
	suite.Suite
	floatStore kvstore.KvStore[float64]
	itemStore  kvstore.KvStore[item]
}

func (s *InMemoryKvStoreTestSuite) SetupTest() {
	s.floatStore = kvstore.NewInMemoryKvStoreWithInterfaces[float64](&kvstore.Settings{
		ModelId: mdl.ModelId{
			Name: "",
		},
		Ttl:       time.Hour,
		BatchSize: 100,
	})
	s.itemStore = kvstore.NewInMemoryKvStoreWithInterfaces[item](&kvstore.Settings{
		ModelId: mdl.ModelId{
			Name: "",
		},
		Ttl:       time.Hour,
		BatchSize: 100,
	})
}

func (s *InMemoryKvStoreTestSuite) TestStoreBasic() {
	ctx := s.T().Context()

	err := s.floatStore.Put(ctx, "key", 1.1)
	s.NoError(err, "there should be no error on Put")

	var v1 float64
	ok, err := s.floatStore.Get(ctx, "key", &v1)
	s.NoError(err, "there should be no error on Get")
	s.True(ok, "the item should be in the store")

	v2 := 1.2
	err = s.floatStore.Put(ctx, "key", v2)
	s.NoError(err, "there should be no error on Put")

	var v3 float64
	ok, err = s.floatStore.Get(ctx, "key", &v3)
	s.NoError(err, "there should be no error on Get")
	s.True(ok, "the item should be in the store")
}

func (s *InMemoryKvStoreTestSuite) TestStoreStruct() {
	ctx := s.T().Context()

	a := item{i: 1, s: "s"}
	err := s.itemStore.Put(ctx, "key", a)
	s.NoError(err, "there should be no error on Put")

	ok, err := s.itemStore.Contains(ctx, "key")
	s.NoError(err, "there should be no error on Contains")
	s.True(ok, "the item should be in the store")

	ok, err = s.itemStore.Contains(ctx, "missing")
	s.NoError(err, "there should be no error on Contains")
	s.False(ok, "the item should be missing the store")

	aCpy := item{}
	ok, err = s.itemStore.Get(ctx, "key", &aCpy)
	s.NoError(err, "there should be no error on Get")
	s.True(ok, "the item should be in the store")
	s.Equal(a, aCpy, "the retrieved item should match the original one")

	putBatch := map[string]item{
		"b": {i: 2, s: "t"},
		"c": {i: 3, s: "u"},
	}
	err = s.itemStore.PutBatch(ctx, putBatch)
	s.NoError(err, "there should be no error on PutBatch")

	getBatch := make(map[string]item, 3)
	missing, err := s.itemStore.GetBatch(ctx, []string{"b", "c", "d"}, getBatch)
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
