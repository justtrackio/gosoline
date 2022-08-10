//go:build integration

package ddb_test

import (
	"context"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/ddb"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/test/suite"
)

type TestData struct {
	Id   string `json:"id" ddb:"key=hash"`
	Data string `json:"data"`
	Ttl  int64  `json:"ttl" ddb:"ttl=enabled"`
}

type DdbTestSuite struct {
	suite.Suite
	repo  ddb.Repository
	clock clock.FakeClock
}

func (s *DdbTestSuite) SetupSuite() []suite.Option {
	s.clock = clock.NewFakeClockAt(time.Now().UTC())

	return []suite.Option{
		suite.WithLogLevel("debug"),
		suite.WithClockProvider(s.clock),
		suite.WithConfigFile("./config.dist.yml"),
	}
}

func (s *DdbTestSuite) SetupTest() error {
	ddbConfig := &ddb.Settings{
		ModelId: mdl.ModelId{
			Name: "test-data",
		},
		Main: ddb.MainSettings{
			Model:              &TestData{},
			ReadCapacityUnits:  1,
			WriteCapacityUnits: 1,
		},
	}
	var err error
	s.repo, err = ddb.NewRepository(s.Env().Context(), s.Env().Config(), s.Env().Logger(), ddbConfig)
	if err != nil {
		return err
	}

	return nil
}

func (s *DdbTestSuite) TestWriteReadUpdateDeleteItem() {
	ctx := context.Background()

	_, err := s.repo.PutItem(ctx, nil, s.makeItem("1", "abc", time.Hour))
	s.NoError(err)

	item := &TestData{}
	result, err := s.repo.GetItem(ctx, s.repo.GetItemBuilder().WithHash("1"), item)
	s.NoError(err)
	s.True(result.IsFound)
	s.Equal(s.makeItem("1", "abc", time.Hour), item)

	_, err = s.repo.UpdateItem(ctx, s.repo.UpdateItemBuilder().WithHash("1").Set("data", "def"), nil)
	s.NoError(err)

	item = &TestData{}
	result, err = s.repo.GetItem(ctx, s.repo.GetItemBuilder().WithHash("1"), item)
	s.NoError(err)
	s.True(result.IsFound)
	s.Equal(s.makeItem("1", "def", time.Hour), item)

	_, err = s.repo.DeleteItem(ctx, s.repo.DeleteItemBuilder().WithHash("1"), &TestData{})
	s.NoError(err)

	item = &TestData{}
	result, err = s.repo.GetItem(ctx, s.repo.GetItemBuilder().WithHash("1"), item)
	s.NoError(err)
	s.False(result.IsFound)
}

func (s *DdbTestSuite) TestGetExpiredItem() {
	ctx := context.Background()

	_, err := s.repo.PutItem(ctx, nil, s.makeItem("1", "abc", time.Second*10))
	s.NoError(err)

	// expire the record for us (ddb will still carry them)
	s.clock.Advance(time.Minute)

	item := &TestData{}
	result, err := s.repo.GetItem(ctx, s.repo.GetItemBuilder().WithHash("1"), item)
	s.NoError(err)
	s.False(result.IsFound)
}

func (s *DdbTestSuite) TestBatch() {
	ctx := context.Background()

	_, err := s.repo.BatchPutItems(ctx, []*TestData{
		s.makeItem("1", "abc", time.Hour),
		s.makeItem("2", "def", time.Second*10),
		s.makeItem("3", "xyz", time.Hour),
	})
	s.NoError(err)

	var items []*TestData
	_, err = s.repo.BatchGetItems(ctx, s.repo.BatchGetItemsBuilder().WithHashKeys([]string{"1", "2"}), &items)
	s.NoError(err)
	s.Equal(len(items), 2)
	if items[0].Id == "2" {
		items[0], items[1] = items[1], items[0]
	}
	s.Equal([]*TestData{
		s.makeItem("1", "abc", time.Hour),
		s.makeItem("2", "def", time.Second*10),
	}, items)

	// expire one record
	s.clock.Advance(time.Minute)

	items = make([]*TestData, 0)
	_, err = s.repo.BatchGetItems(ctx, s.repo.BatchGetItemsBuilder().WithHashKeys([]string{"1", "2"}), &items)
	s.NoError(err)
	s.Equal([]*TestData{
		s.makeItem("1", "abc", time.Hour-time.Minute),
	}, items)

	_, err = s.repo.BatchDeleteItems(ctx, []*TestData{
		s.makeItem("1", "abc", time.Hour),
		s.makeItem("2", "def", time.Second*10),
		s.makeItem("3", "xyz", time.Hour),
	})
	s.NoError(err)

	items = make([]*TestData, 0)
	_, err = s.repo.BatchGetItems(ctx, s.repo.BatchGetItemsBuilder().WithHashKeys([]string{"1", "2", "3"}), &items)
	s.NoError(err)
	s.Equal(len(items), 0)
}

func (s *DdbTestSuite) TestScan() {
	ctx := context.Background()

	_, err := s.repo.PutItem(ctx, nil, s.makeItem("1", "abc", time.Hour))
	s.NoError(err)

	_, err = s.repo.PutItem(ctx, nil, s.makeItem("2", "def", time.Second*10))
	s.NoError(err)

	// expire the records for us (ddb will still carry them)
	s.clock.Advance(time.Minute)

	var items []*TestData
	result, err := s.repo.Scan(ctx, s.repo.ScanBuilder(), &items)
	s.NoError(err)
	s.Equal(int32(1), result.ItemCount)
	s.Equal([]*TestData{
		s.makeItem("1", "abc", time.Hour-time.Minute),
	}, items)
}

func (s *DdbTestSuite) TestQuery() {
	ctx := context.Background()

	_, err := s.repo.PutItem(ctx, nil, s.makeItem("1", "abc", time.Hour))
	s.NoError(err)

	_, err = s.repo.PutItem(ctx, nil, s.makeItem("2", "def", time.Second*10))
	s.NoError(err)

	// expire the records for us (ddb will still carry them)
	s.clock.Advance(time.Minute)

	var items []*TestData
	result, err := s.repo.Query(ctx, s.repo.QueryBuilder().WithHash("1"), &items)
	s.NoError(err)
	s.Equal(int32(1), result.ItemCount)
	s.Equal([]*TestData{
		s.makeItem("1", "abc", time.Hour-time.Minute),
	}, items)

	items = make([]*TestData, 0)
	result, err = s.repo.Query(ctx, s.repo.QueryBuilder().WithHash("2"), &items)
	s.NoError(err)
	s.Equal(int32(0), result.ItemCount)
	s.Equal(0, len(items))
}

func (s *DdbTestSuite) makeItem(id string, data string, ttl time.Duration) *TestData {
	return &TestData{
		Id:   id,
		Data: data,
		Ttl:  s.clock.Now().Add(ttl).Unix(),
	}
}

func TestDdb(t *testing.T) {
	suite.Run(t, new(DdbTestSuite))
}
