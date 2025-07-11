package redis_test

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/elliotchance/redismock/v8"
	baseRedis "github.com/go-redis/redis/v8"
	"github.com/justtrackio/gosoline/pkg/exec"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/redis"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type ClientWithMiniRedisTestSuite struct {
	suite.Suite

	settings   *redis.Settings
	server     *miniredis.Miniredis
	baseClient *baseRedis.Client
	client     redis.Client
}

func (s *ClientWithMiniRedisTestSuite) SetupTest() {
	server, err := miniredis.Run()
	if err != nil {
		s.FailNow(err.Error(), "can not start miniredis")

		return
	}

	s.settings = &redis.Settings{}
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(s.T()))
	executor := exec.NewDefaultExecutor()

	s.baseClient = baseRedis.NewClient(&baseRedis.Options{
		Addr: server.Addr(),
	})

	s.server = server
	s.client = redis.NewClientWithInterfaces(logger, s.baseClient, executor, s.settings)
}

func (s *ClientWithMiniRedisTestSuite) TestGetNotFound() {
	// the logger should fail the test as soon as any logger.Warn or anything gets called
	// because we want to test the executor not doing that
	logger := logMocks.NewLogger(s.T())
	logger.EXPECT().WithFields(mock.AnythingOfType("log.Fields")).Return(logger).Once()
	logger.EXPECT().WithContext(s.T().Context()).Return(logger).Once()
	executor := redis.NewBackoffExecutor(logger, exec.BackoffSettings{
		CancelDelay:     time.Second,
		InitialInterval: time.Millisecond,
		MaxAttempts:     0,
		MaxInterval:     time.Second * 3,
		MaxElapsedTime:  0,
	}, "test")
	s.client = redis.NewClientWithInterfaces(logger, s.baseClient, executor, s.settings)

	res, err := s.client.Get(s.T().Context(), "missing")

	s.Equal(redis.Nil, err)
	s.Equal("", res)
}

func (s *ClientWithMiniRedisTestSuite) TestBLPop() {
	if _, err := s.server.Lpush("list", "value"); err != nil {
		s.FailNow(err.Error(), "can not setup miniredis server")
	}

	res, err := s.client.BLPop(s.T().Context(), 1*time.Second, "list")

	s.NoError(err, "there should be no error on blpop")
	s.Equal("value", res[1])
}

func (s *ClientWithMiniRedisTestSuite) TestDel() {
	count, err := s.client.Del(s.T().Context(), "test")
	s.NoError(err, "there should be no error on Del")
	s.Equal(0, int(count))

	var ttl time.Duration
	err = s.client.Set(s.T().Context(), "key", "value", ttl)
	s.NoError(err, "there should be no error on Del")

	count, err = s.client.Del(s.T().Context(), "key")
	s.NoError(err, "there should be no error on Del")
	s.Equal(1, int(count))
}

func (s *ClientWithMiniRedisTestSuite) TestLLen() {
	for i := 0; i < 3; i++ {
		if _, err := s.server.Lpush("list", "value"); err != nil {
			s.FailNow(err.Error(), "can not setup miniredis server")
		}
	}

	res, err := s.client.LLen(s.T().Context(), "list")

	s.NoError(err, "there should be no error on LLen")
	s.Equal(int64(3), res)
}

func (s *ClientWithMiniRedisTestSuite) TestRPush() {
	count, err := s.client.RPush(s.T().Context(), "list", "v1", "v2", "v3")
	s.NoError(err, "there should be no error on RPush")
	s.Equal(int64(3), count)
}

func (s *ClientWithMiniRedisTestSuite) TestSet() {
	var ttl time.Duration
	err := s.client.Set(s.T().Context(), "key", "value", ttl)
	s.NoError(err, "there should be no error on Set")

	ttl, err = time.ParseDuration("1m")
	s.NoError(err, "there should be no error on ParseDuration")

	err = s.client.Set(s.T().Context(), "key", "value", ttl)
	s.NoError(err, "there should be no error on Set with expiration date")
}

func (s *ClientWithMiniRedisTestSuite) TestHSet() {
	err := s.client.HSet(s.T().Context(), "key", "field", "value")
	s.NoError(err, "there should be no error on HSet")
}

func (s *ClientWithMiniRedisTestSuite) TestHSetNX() {
	isNewlySet, err := s.client.HSetNX(s.T().Context(), "key", "field", "value")
	s.True(isNewlySet, "the field should be set the first time")
	s.NoError(err, "there should be no error on HSet")

	isNewlySet, err = s.client.HSetNX(s.T().Context(), "key", "field", "value")
	s.False(isNewlySet, "the field should NOT be set the first time")
	s.NoError(err, "there should be no error on HSet")
}

func (s *ClientWithMiniRedisTestSuite) TestHMSet() {
	err := s.client.HMSet(s.T().Context(), "key", map[string]any{"field": "value"})
	s.NoError(err, "there should be no error on HSet")
}

func (s *ClientWithMiniRedisTestSuite) TestHMGet() {
	vals, err := s.client.HMGet(s.T().Context(), "key", "field", "value")
	s.NoError(err, "there should be no error on HSet")
	s.Equal([]any{nil, nil}, vals, "there should be no error on HSet")

	err = s.client.HMSet(s.T().Context(), "key", map[string]any{"value": "1"})
	s.NoError(err, "there should be no error on HSet")

	vals, err = s.client.HMGet(s.T().Context(), "key", "field", "value")
	s.NoError(err, "there should be no error on HSet")
	s.Equal([]any{nil, "1"}, vals, "there should be no error on HSet")
}

func (s *ClientWithMiniRedisTestSuite) TestHGetAll() {
	err := s.client.HSet(s.T().Context(), "key", "field1", "value1")
	s.NoError(err, "there should be no error on HSet")
	serr := s.client.HSet(s.T().Context(), "key", "field2", "value2")
	s.NoError(serr, "there should be no error on HSet")

	vals, err := s.client.HGetAll(s.T().Context(), "key")
	s.NoError(err, "there should be no error on HGetAll")
	s.Equal(map[string]string{"field1": "value1", "field2": "value2"}, vals)
}

func (s *ClientWithMiniRedisTestSuite) TestGetDel() {
	var ttl time.Duration
	err := s.client.Set(s.T().Context(), "key", "value1", ttl)
	s.NoError(err, "there should be no error on HSet")

	val, err := s.client.GetDel(s.T().Context(), "key")
	s.NoError(err, "there should be no error on HGetAll")
	s.Equal("value1", val)

	_, err = s.client.GetDel(s.T().Context(), "key")
	s.Equal(redis.Nil, err)
}

func (s *ClientWithMiniRedisTestSuite) TestGetSet() {
	val, err := s.client.GetSet(s.T().Context(), "key", "value1")
	s.Equal(redis.Nil, err)
	s.Equal("", val)

	val, err = s.client.GetSet(s.T().Context(), "key", "value2")
	s.NoError(err, "there should be no error on GetSet")
	s.Equal("value1", val)
}

func (s *ClientWithMiniRedisTestSuite) TestHDel() {
	err := s.client.HSet(s.T().Context(), "key", "field", "value1")
	s.NoError(err, "there should be no error on HSet")
	serr := s.client.HSet(s.T().Context(), "key", "field2", "value2")
	s.NoError(serr, "there should be no error on HSet")

	vals, err := s.client.HDel(s.T().Context(), "key", "field2")
	s.NoError(err, "there should be no error on HDel")
	s.Equal(int64(1), vals)

	valuesFromMap, err := s.client.HGetAll(s.T().Context(), "key")
	s.NoError(err, "there should be no error on HGetAll")
	s.Equal(map[string]string{"field": "value1"}, valuesFromMap)
}

func (s *ClientWithMiniRedisTestSuite) TestSAdd() {
	_, err := s.client.SAdd(s.T().Context(), "key", "value")
	s.NoError(err, "there should be no error on SAdd")
}

func (s *ClientWithMiniRedisTestSuite) TestSCard() {
	_, err := s.client.SAdd(s.T().Context(), "key", "value1")
	s.NoError(err, "there should be no error on SAdd")
	_, err = s.client.SAdd(s.T().Context(), "key", "value2")
	s.NoError(err, "there should be no error on SAdd")
	_, err = s.client.SAdd(s.T().Context(), "key", "value2")
	s.NoError(err, "there should be no error on SAdd")

	amount, err := s.client.SCard(s.T().Context(), "key")
	s.Equal(int64(2), amount)
	s.NoError(err, "there should be no error on SCard")
}

func (s *ClientWithMiniRedisTestSuite) TestSDiff() {
	_, err := s.client.SAdd(s.T().Context(), "set1", "1", "2", "3", "4")
	s.NoError(err, "there should be no error on SAdd")
	_, err = s.client.SAdd(s.T().Context(), "set2", "2", "3")
	s.NoError(err, "there should be no error on SAdd")
	_, err = s.client.SAdd(s.T().Context(), "set3", "2", "4")
	s.NoError(err, "there should be no error on SAdd")

	diffs, err := s.client.SDiff(s.T().Context(), "set1", "set2", "set3")
	s.Equal(1, len(diffs))
	s.True(slices.Contains(diffs, "1"))
	s.NoError(err, "there should be no error on SDiff")
}

func (s *ClientWithMiniRedisTestSuite) TestSDiffStore() {
	_, err := s.client.SAdd(s.T().Context(), "set1", "1", "2", "3", "4")
	s.NoError(err, "there should be no error on SAdd")
	_, err = s.client.SAdd(s.T().Context(), "set2", "2", "3")
	s.NoError(err, "there should be no error on SAdd")
	_, err = s.client.SAdd(s.T().Context(), "set3", "2", "4")
	s.NoError(err, "there should be no error on SAdd")

	numDiffs, err := s.client.SDiffStore(s.T().Context(), "set4", "set1", "set2", "set3")
	s.Equal(int64(1), numDiffs)
	s.NoError(err, "there should be no error on SDiffStore")
}

func (s *ClientWithMiniRedisTestSuite) TestSInter() {
	_, err := s.client.SAdd(s.T().Context(), "set1", "1", "2", "3", "4")
	s.NoError(err, "there should be no error on SAdd")
	_, err = s.client.SAdd(s.T().Context(), "set2", "2", "3")
	s.NoError(err, "there should be no error on SAdd")
	_, err = s.client.SAdd(s.T().Context(), "set3", "2", "4")
	s.NoError(err, "there should be no error on SAdd")

	inters, err := s.client.SInter(s.T().Context(), "set1", "set2", "set3")
	s.Equal(1, len(inters))
	s.True(slices.Contains(inters, "2"))
	s.NoError(err, "there should be no error on SInter")
}

func (s *ClientWithMiniRedisTestSuite) TestSInterStore() {
	_, err := s.client.SAdd(s.T().Context(), "set1", "1", "2", "3", "4")
	s.NoError(err, "there should be no error on SAdd")
	_, err = s.client.SAdd(s.T().Context(), "set2", "2", "3")
	s.NoError(err, "there should be no error on SAdd")
	_, err = s.client.SAdd(s.T().Context(), "set3", "2", "4")
	s.NoError(err, "there should be no error on SAdd")

	numDiffs, err := s.client.SInterStore(s.T().Context(), "set4", "set1", "set2", "set3")
	s.Equal(int64(1), numDiffs)
	s.NoError(err, "there should be no error on SInterStore")
}

func (s *ClientWithMiniRedisTestSuite) TestSMembers() {
	_, err := s.client.SAdd(s.T().Context(), "set", "0", "1", "2")
	s.NoError(err, "there should be no error on SAdd")

	members, err := s.client.SMembers(s.T().Context(), "set")
	s.Equal(3, len(members))
	s.True(slices.Contains(members, "0"))
	s.True(slices.Contains(members, "1"))
	s.True(slices.Contains(members, "2"))
	s.NoError(err, "there should be no error on SMembers")
}

func (s *ClientWithMiniRedisTestSuite) TestSIsMember() {
	_, err := s.client.SAdd(s.T().Context(), "key", "value1")
	s.NoError(err, "there should be no error on SAdd")

	isMember, err := s.client.SIsMember(s.T().Context(), "key", "value1")
	s.Equal(true, isMember)
	s.NoError(err, "there should be no error on SIsMember")

	isMember, err = s.client.SIsMember(s.T().Context(), "key", "value2")
	s.Equal(false, isMember)
	s.NoError(err, "there should be no error on SIsMember")
}

func (s *ClientWithMiniRedisTestSuite) TestSMove() {
	_, err := s.client.SAdd(s.T().Context(), "set1", "0", "1", "2")
	s.NoError(err, "there should be no error on SAdd")

	_, err = s.client.SMove(s.T().Context(), "set1", "set2", "1")
	s.NoError(err, "there should be no error on SMove")

	members, err := s.client.SMembers(s.T().Context(), "set2")
	s.Equal(1, len(members))
	s.True(slices.Contains(members, "1"))
	s.NoError(err, "there should be no error on SMembers")
}

func (s *ClientWithMiniRedisTestSuite) TestSPop() {
	_, err := s.client.SAdd(s.T().Context(), "set1", "0", "1", "2")
	s.NoError(err, "there should be no error on SAdd")

	_, err = s.client.SPop(s.T().Context(), "set1")
	s.NoError(err, "there should be no error on SPop")
}

func (s *ClientWithMiniRedisTestSuite) TestSRem() {
	count, err := s.client.SAdd(s.T().Context(), "set", "v1", "v2", "v3")
	s.NoError(err, "there should be no error on SAdd")
	s.Equal(int64(3), count)

	numRemoved, err := s.client.SRem(s.T().Context(), "set", "v1", "v3")
	s.NoError(err, "there should be no error on SRem")
	s.Equal(int64(2), numRemoved)
}

func (s *ClientWithMiniRedisTestSuite) TestSRandMember() {
	count, err := s.client.SAdd(s.T().Context(), "set", "v1", "v2", "v3")
	s.NoError(err, "there should be no error on SAdd")
	s.Equal(int64(3), count)

	item, err := s.client.SRandMember(s.T().Context(), "set")
	s.NoError(err, "there should be no error on SRandMember")

	contained, err := s.client.SIsMember(s.T().Context(), "set", item)
	s.Equal(true, contained)
	s.NoError(err, "there should be no error on SIsMember")
}

func (s *ClientWithMiniRedisTestSuite) TestSUnion() {
	_, err := s.client.SAdd(s.T().Context(), "set1", "1", "2", "3", "4")
	s.NoError(err, "there should be no error on SAdd")
	_, err = s.client.SAdd(s.T().Context(), "set2", "2", "3")
	s.NoError(err, "there should be no error on SAdd")
	_, err = s.client.SAdd(s.T().Context(), "set3", "2", "4")
	s.NoError(err, "there should be no error on SAdd")

	union, err := s.client.SUnion(s.T().Context(), "set1", "set2", "set3")
	s.Equal(4, len(union))
	s.NoError(err, "there should be no error on SUnion")
}

func (s *ClientWithMiniRedisTestSuite) TestSUnionStore() {
	_, err := s.client.SAdd(s.T().Context(), "set1", "1", "2", "3", "4")
	s.NoError(err, "there should be no error on SAdd")
	_, err = s.client.SAdd(s.T().Context(), "set2", "2", "3")
	s.NoError(err, "there should be no error on SAdd")
	_, err = s.client.SAdd(s.T().Context(), "set3", "2", "4")
	s.NoError(err, "there should be no error on SAdd")

	numUnion, err := s.client.SUnionStore(s.T().Context(), "set4", "set1", "set2", "set3")
	s.Equal(int64(4), numUnion)
	s.NoError(err, "there should be no error on SUnionStore")
}

func (s *ClientWithMiniRedisTestSuite) TestIncr() {
	val, err := s.client.Incr(s.T().Context(), "key")
	s.NoError(err, "there should be no error on Incr")
	s.Equal(int64(1), val)

	val, err = s.client.Incr(s.T().Context(), "key")
	s.NoError(err, "there should be no error on Incr")
	s.Equal(int64(2), val)

	val, err = s.client.IncrBy(s.T().Context(), "key", int64(3))
	s.NoError(err, "there should be no error on IncrBy")
	s.Equal(int64(5), val)
}

func (s *ClientWithMiniRedisTestSuite) TestDecr() {
	err := s.client.Set(s.T().Context(), "key", 10, time.Minute*10)
	s.NoError(err, "there should be no error on Set")

	val, err := s.client.Decr(s.T().Context(), "key")
	s.NoError(err, "there should be no error on Decr")
	s.Equal(int64(9), val)

	val, err = s.client.Decr(s.T().Context(), "key")
	s.NoError(err, "there should be no error on Decr")
	s.Equal(int64(8), val)

	val, err = s.client.DecrBy(s.T().Context(), "key", int64(5))
	s.NoError(err, "there should be no error on DecrBy")
	s.Equal(int64(3), val)
}

func (s *ClientWithMiniRedisTestSuite) TestZAdd() {
	inserts, err := s.client.ZAdd(s.T().Context(), "key", 10.0, "member")
	s.NoError(err, "there should be no error on ZAdd")
	s.EqualValues(1, inserts)
}

func (s *ClientWithMiniRedisTestSuite) TestZAddArgs() {
	args := redis.ZAddArgs{
		Key: "key",
		Members: []redis.Z{
			{Member: "member1", Score: 20},
			{Member: "member2", Score: 50},
			{Member: "member3", Score: 110},
			{Member: "member4", Score: 160},
		},
	}
	inserts, err := s.client.ZAddArgs(s.T().Context(), args)
	s.NoError(err, "there should be no error on ZAdd")
	s.EqualValues(4, inserts)
}

func (s *ClientWithMiniRedisTestSuite) TestZCard() {
	cardinality, err := s.client.ZCard(s.T().Context(), "key")
	s.NoError(err, "there should be no error on ZCard")
	s.EqualValues(0, cardinality)

	_, err = s.client.ZAdd(s.T().Context(), "key", 10.0, "member0")
	s.NoError(err, "there should be no error on ZAdd")

	_, err = s.client.ZAdd(s.T().Context(), "key", 10.0, "member1")
	s.NoError(err, "there should be no error on ZAdd")

	_, err = s.client.ZAdd(s.T().Context(), "key", 11.0, "member2")
	s.NoError(err, "there should be no error on ZAdd")

	_, err = s.client.ZAdd(s.T().Context(), "key", 12.0, "member3")
	s.NoError(err, "there should be no error on ZAdd")

	cardinality, err = s.client.ZCard(s.T().Context(), "key")
	s.NoError(err, "there should be no error on ZCard")
	s.EqualValues(4, cardinality)
}

func (s *ClientWithMiniRedisTestSuite) TestZCount() {
	_, err := s.client.ZAdd(s.T().Context(), "key", 10, "member0")
	s.NoError(err)
	_, err = s.client.ZAdd(s.T().Context(), "key", 50, "member1")
	s.NoError(err)
	_, err = s.client.ZAdd(s.T().Context(), "key", 90, "member2")
	s.NoError(err)
	_, err = s.client.ZAdd(s.T().Context(), "key", 100, "member3")
	s.NoError(err)
	_, err = s.client.ZAdd(s.T().Context(), "key", 120, "member4")
	s.NoError(err)

	count, err := s.client.ZCount(s.T().Context(), "key", "(50", "100")
	s.NoError(err, "there should be no error on ZCount")
	s.EqualValues(2, count)
}

func (s *ClientWithMiniRedisTestSuite) TestZIncrBy() {
	_, err := s.client.ZIncrBy(s.T().Context(), "key", 10.0, "member")
	s.NoError(err)
	_, err = s.client.ZIncrBy(s.T().Context(), "key", 50.0, "member")
	s.NoError(err)

	score, err := s.client.ZScore(s.T().Context(), "key", "member")
	s.NoError(err, "there should be no error on ZIncrBy")
	s.EqualValues(60.0, score)
}

func (s *ClientWithMiniRedisTestSuite) TestZScore() {
	_, err := s.client.ZAdd(s.T().Context(), "key", 10.0, "member")
	s.NoError(err, "there should be no error on ZAdd")

	score, err := s.client.ZScore(s.T().Context(), "key", "member")
	s.NoError(err, "there should be no error on ZScore")
	s.EqualValues(10.0, score)
}

func (s *ClientWithMiniRedisTestSuite) TestZRange() {
	_, err := s.client.ZAdd(s.T().Context(), "key", 10, "member0")
	s.NoError(err)
	_, err = s.client.ZAdd(s.T().Context(), "key", 50, "member1")
	s.NoError(err)
	_, err = s.client.ZAdd(s.T().Context(), "key", 90, "member2")
	s.NoError(err)
	_, err = s.client.ZAdd(s.T().Context(), "key", 100, "member3")
	s.NoError(err)
	_, err = s.client.ZAdd(s.T().Context(), "key", 120, "member4")
	s.NoError(err)

	members, err := s.client.ZRange(s.T().Context(), "key", 1, 2)
	s.NoError(err, "there should be no error on ZRange")
	s.EqualValues(2, len(members))
	s.Equal("member1", members[0])
	s.Equal("member2", members[1])
}

func (s *ClientWithMiniRedisTestSuite) TestZRangeArgsWithScore() {
	_, err := s.client.ZAdd(s.T().Context(), "key", 10, "member0")
	s.NoError(err)
	_, err = s.client.ZAdd(s.T().Context(), "key", 50, "member1")
	s.NoError(err)
	_, err = s.client.ZAdd(s.T().Context(), "key", 90, "member2")
	s.NoError(err)
	_, err = s.client.ZAdd(s.T().Context(), "key", 100, "member3")
	s.NoError(err)
	_, err = s.client.ZAdd(s.T().Context(), "key", 120, "member4")
	s.NoError(err)

	args := redis.ZRangeArgs{
		Key:   "key",
		Start: 1,
		Stop:  2,
	}

	zs, err := s.client.ZRangeArgsWithScore(s.T().Context(), args)
	s.NoError(err, "there should be no error on ZRangeArgsWithScore")
	s.EqualValues(2, len(zs))
	s.Equal("member1", zs[0].Member)
	s.Equal(float64(50), zs[0].Score)
	s.Equal("member2", zs[1].Member)
	s.Equal(float64(90), zs[1].Score)
}

func (s *ClientWithMiniRedisTestSuite) TestZRangeArgsWithScoreRev() {
	_, err := s.client.ZAdd(s.T().Context(), "key", 10, "member0")
	s.NoError(err)
	_, err = s.client.ZAdd(s.T().Context(), "key", 50, "member1")
	s.NoError(err)
	_, err = s.client.ZAdd(s.T().Context(), "key", 90, "member2")
	s.NoError(err)
	_, err = s.client.ZAdd(s.T().Context(), "key", 100, "member3")
	s.NoError(err)
	_, err = s.client.ZAdd(s.T().Context(), "key", 120, "member4")
	s.NoError(err)

	args := redis.ZRangeArgs{
		Key:   "key",
		Start: 1,
		Stop:  2,
		Rev:   true,
	}

	zs, err := s.client.ZRangeArgsWithScore(s.T().Context(), args)
	s.NoError(err, "there should be no error on ZRangeArgsWithScore")
	s.EqualValues(2, len(zs))
	s.Equal("member3", zs[0].Member)
	s.Equal(float64(100), zs[0].Score)
	s.Equal("member2", zs[1].Member)
	s.Equal(float64(90), zs[1].Score)
}

func (s *ClientWithMiniRedisTestSuite) TestZRangeArgsByScoreWithLimit() {
	_, err := s.client.ZAdd(s.T().Context(), "key", 10, "member0")
	s.NoError(err)
	_, err = s.client.ZAdd(s.T().Context(), "key", 50, "member1")
	s.NoError(err)
	_, err = s.client.ZAdd(s.T().Context(), "key", 90, "member2")
	s.NoError(err)
	_, err = s.client.ZAdd(s.T().Context(), "key", 100, "member3")
	s.NoError(err)
	_, err = s.client.ZAdd(s.T().Context(), "key", 120, "member4")
	s.NoError(err)
	_, err = s.client.ZAdd(s.T().Context(), "key", 150, "member5")
	s.NoError(err)
	_, err = s.client.ZAdd(s.T().Context(), "key", 170, "member6")
	s.NoError(err)

	args := redis.ZRangeArgs{
		Key:     "key",
		Start:   "(10",
		Stop:    "200",
		Rev:     false,
		ByScore: true,
		Offset:  1,
		Count:   3,
	}

	zs, err := s.client.ZRangeArgs(s.T().Context(), args)
	s.NoError(err, "there should be no error on ZRangeArgs")
	s.EqualValues(3, len(zs))
	s.Equal("member2", zs[0])
	s.Equal("member3", zs[1])
	s.Equal("member4", zs[2])
}

func (s *ClientWithMiniRedisTestSuite) TestZRank() {
	_, err := s.client.ZAdd(s.T().Context(), "key", 10, "member0")
	s.NoError(err)
	_, err = s.client.ZAdd(s.T().Context(), "key", 50, "member1")
	s.NoError(err)
	_, err = s.client.ZAdd(s.T().Context(), "key", 90, "member2")
	s.NoError(err)
	_, err = s.client.ZAdd(s.T().Context(), "key", 100, "member3")
	s.NoError(err)
	_, err = s.client.ZAdd(s.T().Context(), "key", 120, "member4")
	s.NoError(err)

	rank, err := s.client.ZRank(s.T().Context(), "key", "member3")
	s.NoError(err, "there should be no error on ZRank")
	s.EqualValues(3, rank)
}

func (s *ClientWithMiniRedisTestSuite) TestZRem() {
	_, err := s.client.ZAdd(s.T().Context(), "key", 10, "member0")
	s.NoError(err)
	_, err = s.client.ZAdd(s.T().Context(), "key", 50, "member1")
	s.NoError(err)
	_, err = s.client.ZAdd(s.T().Context(), "key", 90, "member2")
	s.NoError(err)
	_, err = s.client.ZAdd(s.T().Context(), "key", 100, "member3")
	s.NoError(err)
	_, err = s.client.ZAdd(s.T().Context(), "key", 120, "member4")
	s.NoError(err)

	removes, err := s.client.ZRem(s.T().Context(), "key", "member2", "member3", "memberX")
	s.NoError(err, "there should be no error on ZRem")
	s.EqualValues(2, removes)
}

func (s *ClientWithMiniRedisTestSuite) TestZRevRank() {
	_, err := s.client.ZAdd(s.T().Context(), "key", 10, "member0")
	s.NoError(err)
	_, err = s.client.ZAdd(s.T().Context(), "key", 50, "member1")
	s.NoError(err)
	_, err = s.client.ZAdd(s.T().Context(), "key", 90, "member2")
	s.NoError(err)
	_, err = s.client.ZAdd(s.T().Context(), "key", 100, "member3")
	s.NoError(err)
	_, err = s.client.ZAdd(s.T().Context(), "key", 120, "member4")
	s.NoError(err)

	rank, err := s.client.ZRevRank(s.T().Context(), "key", "member3")
	s.NoError(err, "there should be no error on ZRevRank")
	s.EqualValues(1, rank)
}

func (s *ClientWithMiniRedisTestSuite) TestZRandMember() {
	_, err := s.client.ZAdd(s.T().Context(), "key", 10, "member0")
	s.NoError(err)
	_, err = s.client.ZAdd(s.T().Context(), "key", 50, "member1")
	s.NoError(err)
	_, err = s.client.ZAdd(s.T().Context(), "key", 90, "member2")
	s.NoError(err)
	_, err = s.client.ZAdd(s.T().Context(), "key", 100, "member3")
	s.NoError(err)
	_, err = s.client.ZAdd(s.T().Context(), "key", 120, "member4")
	s.NoError(err)

	members, err := s.client.ZRandMember(s.T().Context(), "key", 2)
	s.NoError(err, "there should be no error on ZRandMember")
	s.EqualValues(2, len(members))
}

func (s *ClientWithMiniRedisTestSuite) TestLPush() {
	count, err := s.client.LPush(s.T().Context(), "list", "v1", "v2", "v3")
	s.NoError(err, "there should be no error on LPush")
	s.Equal(int64(3), count)
}

func (s *ClientWithMiniRedisTestSuite) TestLRem() {
	count, err := s.client.LPush(s.T().Context(), "list", "v1", "v2", "v3")
	s.NoError(err, "there should be no error on LPush")
	s.Equal(int64(3), count)

	numRemoved, err := s.client.LRem(s.T().Context(), "list", 0, "v1")
	s.NoError(err, "there should be no error on LRem")
	s.Equal(int64(1), numRemoved)
}

func (s *ClientWithMiniRedisTestSuite) TestLPop() {
	count, err := s.client.RPush(s.T().Context(), "list", "v1", "v2", "v3")
	s.NoError(err, "there should be no error on RPush")
	s.Equal(int64(3), count)

	item, err := s.client.LPop(s.T().Context(), "list")
	s.NoError(err, "there should be no error on LPop")
	s.Equal("v1", item)
}

func (s *ClientWithMiniRedisTestSuite) TestRPop() {
	count, err := s.client.LPush(s.T().Context(), "list", "v1", "v2", "v3")
	s.NoError(err, "there should be no error on LPush")
	s.Equal(int64(3), count)

	item, err := s.client.RPop(s.T().Context(), "list")
	s.NoError(err, "there should be no error on RPop")
	s.Equal("v1", item)
}

func (s *ClientWithMiniRedisTestSuite) TestExpire() {
	_, err := s.client.Incr(s.T().Context(), "key")
	s.NoError(err)

	result, err := s.client.Expire(s.T().Context(), "key", time.Second)
	s.NoError(err, "there should be no error on Expire")
	s.True(result)

	s.server.FastForward(time.Second)

	amount, err := s.client.Exists(s.T().Context(), "key")
	s.Equal(int64(0), amount)
	s.NoError(err, "there should be no error on Exists")
}

func (s *ClientWithMiniRedisTestSuite) TestIsAlive() {
	alive := s.client.IsAlive(s.T().Context())
	s.True(alive)
}

func TestClientWithMiniRedisTestSuite(t *testing.T) {
	suite.Run(t, new(ClientWithMiniRedisTestSuite))
}

type ClientWithMockTestSuite struct {
	suite.Suite
	client    redis.Client
	redisMock *redismock.ClientMock
}

func (s *ClientWithMockTestSuite) SetupTest() {
	settings := &redis.Settings{}
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(s.T()))
	executor := redis.NewBackoffExecutor(logger, settings.BackoffSettings, "test")

	s.redisMock = redismock.NewMock()
	s.client = redis.NewClientWithInterfaces(logger, s.redisMock, executor, settings)
}

func (s *ClientWithMockTestSuite) TestSetWithOOM() {
	s.redisMock.On("Set", s.T().Context(), "key", "value", time.Duration(1000000000)).Return(baseRedis.NewStatusResult("", errors.New("OOM command not allowed when used memory > 'maxmemory'"))).Once()
	s.redisMock.On("Set", s.T().Context(), "key", "value", time.Duration(1000000000)).Return(baseRedis.NewStatusResult("", nil)).Once()

	err := s.client.Set(s.T().Context(), "key", "value", time.Second)

	s.NoError(err, "there should be no error on Set with backoff")
	s.redisMock.AssertExpectations(s.T())
}

func (s *ClientWithMockTestSuite) TestSetWithError() {
	s.redisMock.On("Set", s.T().Context(), "key", "value", time.Duration(1000000000)).Return(baseRedis.NewStatusResult("", errors.New("random redis error"))).Once()
	s.redisMock.On("Set", s.T().Context(), "key", "value", time.Duration(1000000000)).Return(baseRedis.NewStatusResult("", nil)).Times(0)

	err := s.client.Set(s.T().Context(), "key", "value", time.Second)

	s.NotNil(err, "there should be an error on Set")
	s.redisMock.AssertExpectations(s.T())
}

func (s *ClientWithMockTestSuite) TestPFAddCountMerge() {
	// miniredis doesn't support HyperLogLogs, so we need to mock these
	s.redisMock.On("PFAdd", s.T().Context(), "key1", []any{"a", "b", "c"}).Return(baseRedis.NewIntResult(1, nil)).Once()
	s.redisMock.On("PFAdd", s.T().Context(), "key2", []any{"d"}).Return(baseRedis.NewIntResult(1, nil)).Once()
	s.redisMock.On("PFAdd", s.T().Context(), "key2", []any{"e", "f"}).Return(baseRedis.NewIntResult(1, nil)).Once()
	s.redisMock.On("PFMerge", s.T().Context(), "key", []string{"key1", "key2"}).Return(baseRedis.NewStatusResult("OK", nil)).Once()
	s.redisMock.On("PFCount", s.T().Context(), []string{"key"}).Return(baseRedis.NewIntResult(6, nil)).Once()

	result, err := s.client.PFAdd(s.T().Context(), "key1", "a", "b", "c")
	s.NoError(err, "There should be no error on PFAdd")
	s.Equal(int64(1), result)

	result, err = s.client.PFAdd(s.T().Context(), "key2", "d")
	s.NoError(err, "There should be no error on PFAdd")
	s.Equal(int64(1), result)

	result, err = s.client.PFAdd(s.T().Context(), "key2", "e", "f")
	s.NoError(err, "There should be no error on PFAdd")
	s.Equal(int64(1), result)

	_, err = s.client.PFMerge(s.T().Context(), "key", "key1", "key2")
	s.NoError(err, "There should be no error on PFMerge")

	result, err = s.client.PFCount(s.T().Context(), "key")
	s.NoError(err, "There should be no error on PFCount")
	s.Equal(int64(6), result)
}

func TestClientWithMockTestSuite(t *testing.T) {
	suite.Run(t, new(ClientWithMockTestSuite))
}
