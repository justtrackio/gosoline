package redis_test

import (
	"context"
	"errors"
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
	logger := logMocks.NewLoggerMockedAll()
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
	logger := new(logMocks.Logger)
	logger.On("WithFields", mock.Anything).Return(logger).Once()
	logger.On("WithContext", context.Background()).Return(logger).Once()
	executor := redis.NewBackoffExecutor(logger, exec.BackoffSettings{
		CancelDelay:     time.Second,
		InitialInterval: time.Millisecond,
		MaxAttempts:     0,
		MaxInterval:     time.Second * 3,
		MaxElapsedTime:  0,
	}, "test")
	s.client = redis.NewClientWithInterfaces(logger, s.baseClient, executor, s.settings)

	res, err := s.client.Get(context.Background(), "missing")

	s.Equal(redis.Nil, err)
	s.Equal("", res)
}

func (s *ClientWithMiniRedisTestSuite) TestBLPop() {
	if _, err := s.server.Lpush("list", "value"); err != nil {
		s.FailNow(err.Error(), "can not setup miniredis server")
	}

	res, err := s.client.BLPop(context.Background(), 1*time.Second, "list")

	s.NoError(err, "there should be no error on blpop")
	s.Equal("value", res[1])
}

func (s *ClientWithMiniRedisTestSuite) TestDel() {
	count, err := s.client.Del(context.Background(), "test")
	s.NoError(err, "there should be no error on Del")
	s.Equal(0, int(count))

	var ttl time.Duration
	err = s.client.Set(context.Background(), "key", "value", ttl)
	s.NoError(err, "there should be no error on Del")

	count, err = s.client.Del(context.Background(), "key")
	s.NoError(err, "there should be no error on Del")
	s.Equal(1, int(count))
}

func (s *ClientWithMiniRedisTestSuite) TestLLen() {
	for i := 0; i < 3; i++ {
		if _, err := s.server.Lpush("list", "value"); err != nil {
			s.FailNow(err.Error(), "can not setup miniredis server")
		}
	}

	res, err := s.client.LLen(context.Background(), "list")

	s.NoError(err, "there should be no error on LLen")
	s.Equal(int64(3), res)
}

func (s *ClientWithMiniRedisTestSuite) TestRPush() {
	count, err := s.client.RPush(context.Background(), "list", "v1", "v2", "v3")
	s.NoError(err, "there should be no error on RPush")
	s.Equal(int64(3), count)
}

func (s *ClientWithMiniRedisTestSuite) TestSet() {
	var ttl time.Duration
	err := s.client.Set(context.Background(), "key", "value", ttl)
	s.NoError(err, "there should be no error on Set")

	ttl, _ = time.ParseDuration("1m")
	err = s.client.Set(context.Background(), "key", "value", ttl)
	s.NoError(err, "there should be no error on Set with expiration date")
}

func (s *ClientWithMiniRedisTestSuite) TestHSet() {
	err := s.client.HSet(context.Background(), "key", "field", "value")
	s.NoError(err, "there should be no error on HSet")
}

func (s *ClientWithMiniRedisTestSuite) TestHSetNX() {
	isNewlySet, err := s.client.HSetNX(context.Background(), "key", "field", "value")
	s.True(isNewlySet, "the field should be set the first time")
	s.NoError(err, "there should be no error on HSet")

	isNewlySet, err = s.client.HSetNX(context.Background(), "key", "field", "value")
	s.False(isNewlySet, "the field should NOT be set the first time")
	s.NoError(err, "there should be no error on HSet")
}

func (s *ClientWithMiniRedisTestSuite) TestHMSet() {
	err := s.client.HMSet(context.Background(), "key", map[string]interface{}{"field": "value"})
	s.NoError(err, "there should be no error on HSet")
}

func (s *ClientWithMiniRedisTestSuite) TestHMGet() {
	vals, err := s.client.HMGet(context.Background(), "key", "field", "value")
	s.NoError(err, "there should be no error on HSet")
	s.Equal([]interface{}{nil, nil}, vals, "there should be no error on HSet")

	err = s.client.HMSet(context.Background(), "key", map[string]interface{}{"value": "1"})
	s.NoError(err, "there should be no error on HSet")

	vals, err = s.client.HMGet(context.Background(), "key", "field", "value")
	s.NoError(err, "there should be no error on HSet")
	s.Equal([]interface{}{nil, "1"}, vals, "there should be no error on HSet")
}

func (s *ClientWithMiniRedisTestSuite) TestHGetAll() {
	err := s.client.HSet(context.Background(), "key", "field1", "value1")
	s.NoError(err, "there should be no error on HSet")
	serr := s.client.HSet(context.Background(), "key", "field2", "value2")
	s.NoError(serr, "there should be no error on HSet")

	vals, err := s.client.HGetAll(context.Background(), "key")
	s.NoError(err, "there should be no error on HGetAll")
	s.Equal(map[string]string{"field1": "value1", "field2": "value2"}, vals)
}

func (s *ClientWithMiniRedisTestSuite) TestGetDel() {
	var ttl time.Duration
	err := s.client.Set(context.Background(), "key", "value1", ttl)
	s.NoError(err, "there should be no error on HSet")

	val, err := s.client.GetDel(context.Background(), "key")
	s.NoError(err, "there should be no error on HGetAll")
	s.Equal("value1", val)

	_, err = s.client.GetDel(context.Background(), "key")
	s.Equal(redis.Nil, err)
}

func (s *ClientWithMiniRedisTestSuite) TestHDel() {
	err := s.client.HSet(context.Background(), "key", "field", "value1")
	s.NoError(err, "there should be no error on HSet")
	serr := s.client.HSet(context.Background(), "key", "field2", "value2")
	s.NoError(serr, "there should be no error on HSet")

	vals, err := s.client.HDel(context.Background(), "key", "field2")
	s.NoError(err, "there should be no error on HDel")
	s.Equal(int64(1), vals)

	valuesFromMap, err := s.client.HGetAll(context.Background(), "key")
	s.NoError(err, "there should be no error on HGetAll")
	s.Equal(map[string]string{"field": "value1"}, valuesFromMap)
}

func (s *ClientWithMiniRedisTestSuite) TestSAdd() {
	_, err := s.client.SAdd(context.Background(), "key", "value")
	s.NoError(err, "there should be no error on SAdd")
}

func (s *ClientWithMiniRedisTestSuite) TestSCard() {
	_, err := s.client.SAdd(context.Background(), "key", "value1")
	s.NoError(err, "there should be no error on SAdd")
	_, err = s.client.SAdd(context.Background(), "key", "value2")
	s.NoError(err, "there should be no error on SAdd")
	_, err = s.client.SAdd(context.Background(), "key", "value2")
	s.NoError(err, "there should be no error on SAdd")

	amount, err := s.client.SCard(context.Background(), "key")
	s.Equal(int64(2), amount)
	s.NoError(err, "there should be no error on SCard")
}

func (s *ClientWithMiniRedisTestSuite) TestSIsMember() {
	_, err := s.client.SAdd(context.Background(), "key", "value1")
	s.NoError(err, "there should be no error on SAdd")

	isMember, err := s.client.SIsMember(context.Background(), "key", "value1")
	s.Equal(true, isMember)
	s.NoError(err, "there should be no error on SIsMember")

	isMember, err = s.client.SIsMember(context.Background(), "key", "value2")
	s.Equal(false, isMember)
	s.NoError(err, "there should be no error on SIsMember")
}

func (s *ClientWithMiniRedisTestSuite) TestIncr() {
	val, err := s.client.Incr(context.Background(), "key")
	s.NoError(err, "there should be no error on Incr")
	s.Equal(int64(1), val)

	val, err = s.client.Incr(context.Background(), "key")
	s.NoError(err, "there should be no error on Incr")
	s.Equal(int64(2), val)

	val, err = s.client.IncrBy(context.Background(), "key", int64(3))
	s.NoError(err, "there should be no error on IncrBy")
	s.Equal(int64(5), val)
}

func (s *ClientWithMiniRedisTestSuite) TestDecr() {
	err := s.client.Set(context.Background(), "key", 10, time.Minute*10)
	s.NoError(err, "there should be no error on Set")

	val, err := s.client.Decr(context.Background(), "key")
	s.NoError(err, "there should be no error on Decr")
	s.Equal(int64(9), val)

	val, err = s.client.Decr(context.Background(), "key")
	s.NoError(err, "there should be no error on Decr")
	s.Equal(int64(8), val)

	val, err = s.client.DecrBy(context.Background(), "key", int64(5))
	s.NoError(err, "there should be no error on DecrBy")
	s.Equal(int64(3), val)
}

func (s *ClientWithMiniRedisTestSuite) TestZAdd() {
	inserts, err := s.client.ZAdd(context.Background(), "key", 10.0, "member")
	s.NoError(err, "there should be no error on ZAdd")
	s.EqualValues(1, inserts)
}

func (s *ClientWithMiniRedisTestSuite) TestZCard() {
	cardinality, err := s.client.ZCard(context.Background(), "key")
	s.NoError(err, "there should be no error on ZCard")
	s.EqualValues(0, cardinality)

	_, err = s.client.ZAdd(context.Background(), "key", 10.0, "member0")
	s.NoError(err, "there should be no error on ZAdd")

	_, err = s.client.ZAdd(context.Background(), "key", 10.0, "member1")
	s.NoError(err, "there should be no error on ZAdd")

	_, err = s.client.ZAdd(context.Background(), "key", 11.0, "member2")
	s.NoError(err, "there should be no error on ZAdd")

	_, err = s.client.ZAdd(context.Background(), "key", 12.0, "member3")
	s.NoError(err, "there should be no error on ZAdd")

	cardinality, err = s.client.ZCard(context.Background(), "key")
	s.NoError(err, "there should be no error on ZCard")
	s.EqualValues(4, cardinality)
}

func (s *ClientWithMiniRedisTestSuite) TestZCount() {
	_, _ = s.client.ZAdd(context.Background(), "key", 10, "member0")
	_, _ = s.client.ZAdd(context.Background(), "key", 50, "member1")
	_, _ = s.client.ZAdd(context.Background(), "key", 90, "member2")
	_, _ = s.client.ZAdd(context.Background(), "key", 100, "member3")
	_, _ = s.client.ZAdd(context.Background(), "key", 120, "member4")

	count, err := s.client.ZCount(context.Background(), "key", "(50", "100")
	s.NoError(err, "there should be no error on ZCount")
	s.EqualValues(2, count)
}

func (s *ClientWithMiniRedisTestSuite) TestZIncrBy() {
	_, _ = s.client.ZIncrBy(context.Background(), "key", 10.0, "member")
	_, _ = s.client.ZIncrBy(context.Background(), "key", 50.0, "member")

	score, err := s.client.ZScore(context.Background(), "key", "member")
	s.NoError(err, "there should be no error on ZIncrBy")
	s.EqualValues(60.0, score)
}

func (s *ClientWithMiniRedisTestSuite) TestZScore() {
	_, _ = s.client.ZAdd(context.Background(), "key", 10.0, "member")

	score, err := s.client.ZScore(context.Background(), "key", "member")
	s.NoError(err, "there should be no error on ZScore")
	s.EqualValues(10.0, score)
}

func (s *ClientWithMiniRedisTestSuite) TestZRange() {
	_, _ = s.client.ZAdd(context.Background(), "key", 10, "member0")
	_, _ = s.client.ZAdd(context.Background(), "key", 50, "member1")
	_, _ = s.client.ZAdd(context.Background(), "key", 90, "member2")
	_, _ = s.client.ZAdd(context.Background(), "key", 100, "member3")
	_, _ = s.client.ZAdd(context.Background(), "key", 120, "member4")

	members, err := s.client.ZRange(context.Background(), "key", 1, 2)
	s.NoError(err, "there should be no error on ZRange")
	s.EqualValues(2, len(members))
	s.Equal("member1", members[0])
	s.Equal("member2", members[1])
}

func (s *ClientWithMiniRedisTestSuite) TestZRank() {
	_, _ = s.client.ZAdd(context.Background(), "key", 10, "member0")
	_, _ = s.client.ZAdd(context.Background(), "key", 50, "member1")
	_, _ = s.client.ZAdd(context.Background(), "key", 90, "member2")
	_, _ = s.client.ZAdd(context.Background(), "key", 100, "member3")
	_, _ = s.client.ZAdd(context.Background(), "key", 120, "member4")

	rank, err := s.client.ZRank(context.Background(), "key", "member3")
	s.NoError(err, "there should be no error on ZRank")
	s.EqualValues(3, rank)
}

func (s *ClientWithMiniRedisTestSuite) TestZRem() {
	_, _ = s.client.ZAdd(context.Background(), "key", 10, "member0")
	_, _ = s.client.ZAdd(context.Background(), "key", 50, "member1")
	_, _ = s.client.ZAdd(context.Background(), "key", 90, "member2")
	_, _ = s.client.ZAdd(context.Background(), "key", 100, "member3")
	_, _ = s.client.ZAdd(context.Background(), "key", 120, "member4")

	removes, err := s.client.ZRem(context.Background(), "key", "member2", "member3", "memberX")
	s.NoError(err, "there should be no error on ZRem")
	s.EqualValues(2, removes)
}

func (s *ClientWithMiniRedisTestSuite) TestZRevRange() {
	_, _ = s.client.ZAdd(context.Background(), "key", 10, "member0")
	_, _ = s.client.ZAdd(context.Background(), "key", 50, "member1")
	_, _ = s.client.ZAdd(context.Background(), "key", 90, "member2")
	_, _ = s.client.ZAdd(context.Background(), "key", 100, "member3")
	_, _ = s.client.ZAdd(context.Background(), "key", 120, "member4")

	members, err := s.client.ZRevRange(context.Background(), "key", 1, 2)
	s.NoError(err, "there should be no error on ZRevRange")
	s.EqualValues(2, len(members))
	s.Equal("member3", members[0])
	s.Equal("member2", members[1])
}

func (s *ClientWithMiniRedisTestSuite) TestZRevRank() {
	_, _ = s.client.ZAdd(context.Background(), "key", 10, "member0")
	_, _ = s.client.ZAdd(context.Background(), "key", 50, "member1")
	_, _ = s.client.ZAdd(context.Background(), "key", 90, "member2")
	_, _ = s.client.ZAdd(context.Background(), "key", 100, "member3")
	_, _ = s.client.ZAdd(context.Background(), "key", 120, "member4")

	rank, err := s.client.ZRevRank(context.Background(), "key", "member3")
	s.NoError(err, "there should be no error on ZRevRank")
	s.EqualValues(1, rank)
}

// Following commented redis operations are not supported by miniredis yet
// so we can't unit-test them.
// Issue: https://github.com/alicebob/miniredis/issues/302
// TODO: Check if miniredis supports these and update

//func (s *ClientWithMiniRedisTestSuite) TestZRandMember() {
//
//  _, _ = s.client.ZAdd(context.Background(), "key", 10, "member0")
//  _, _ = s.client.ZAdd(context.Background(), "key", 50, "member1")
//  _, _ = s.client.ZAdd(context.Background(), "key", 90, "member2")
//  _, _ = s.client.ZAdd(context.Background(), "key", 100, "member3")
//  _, _ = s.client.ZAdd(context.Background(), "key", 120, "member4")
//
//	members, err := s.client.ZRandMember(context.Background(), "key", 2, true)
//	s.NoError(err, "there should be no error on ZRandMember")
//	s.EqualValues(2, len(members))
//}

//func (s *ClientWithMiniRedisTestSuite) TestZMScore() {
//	_, _ = s.client.ZAdd(context.Background(), "key", 10, "member0")
//	_, _ = s.client.ZAdd(context.Background(), "key", 50, "member1")
//
//	scores, err := s.client.ZMScore(context.Background(), "key", "member0", "member1")
//	s.NoError(err, "there should be no error on ZMScore")
//	s.EqualValues(10.0, scores[0])
//	s.EqualValues(50.0, scores[1])
//}

func (s *ClientWithMiniRedisTestSuite) TestExpire() {
	_, _ = s.client.Incr(context.Background(), "key")

	result, err := s.client.Expire(context.Background(), "key", time.Second)
	s.NoError(err, "there should be no error on Expire")
	s.True(result)

	s.server.FastForward(time.Second)

	amount, err := s.client.Exists(context.Background(), "key")
	s.Equal(int64(0), amount)
	s.NoError(err, "there should be no error on Exists")
}

func (s *ClientWithMiniRedisTestSuite) TestIsAlive() {
	alive := s.client.IsAlive(context.Background())
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
	logger := logMocks.NewLoggerMockedAll()
	executor := redis.NewBackoffExecutor(logger, settings.BackoffSettings, "test")

	s.redisMock = redismock.NewMock()
	s.client = redis.NewClientWithInterfaces(logger, s.redisMock, executor, settings)
}

func (s *ClientWithMockTestSuite) TestSetWithOOM() {
	s.redisMock.On("Set", context.Background(), "key", "value", time.Duration(1000000000)).Return(baseRedis.NewStatusResult("", errors.New("OOM command not allowed when used memory > 'maxmemory'"))).Once()
	s.redisMock.On("Set", context.Background(), "key", "value", time.Duration(1000000000)).Return(baseRedis.NewStatusResult("", nil)).Once()

	err := s.client.Set(context.Background(), "key", "value", time.Second)

	s.NoError(err, "there should be no error on Set with backoff")
	s.redisMock.AssertExpectations(s.T())
}

func (s *ClientWithMockTestSuite) TestSetWithError() {
	s.redisMock.On("Set", context.Background(), "key", "value", time.Duration(1000000000)).Return(baseRedis.NewStatusResult("", errors.New("random redis error"))).Once()
	s.redisMock.On("Set", context.Background(), "key", "value", time.Duration(1000000000)).Return(baseRedis.NewStatusResult("", nil)).Times(0)

	err := s.client.Set(context.Background(), "key", "value", time.Second)

	s.NotNil(err, "there should be an error on Set")
	s.redisMock.AssertExpectations(s.T())
}

func (s *ClientWithMockTestSuite) TestPFAddCountMerge() {
	// miniredis doesn't support HyperLogLogs, so we need to mock these
	s.redisMock.On("PFAdd", context.Background(), "key1", []interface{}{"a", "b", "c"}).Return(baseRedis.NewIntResult(1, nil)).Once()
	s.redisMock.On("PFAdd", context.Background(), "key2", []interface{}{"d"}).Return(baseRedis.NewIntResult(1, nil)).Once()
	s.redisMock.On("PFAdd", context.Background(), "key2", []interface{}{"e", "f"}).Return(baseRedis.NewIntResult(1, nil)).Once()
	s.redisMock.On("PFMerge", context.Background(), "key", []string{"key1", "key2"}).Return(baseRedis.NewStatusResult("OK", nil)).Once()
	s.redisMock.On("PFCount", context.Background(), []string{"key"}).Return(baseRedis.NewIntResult(6, nil)).Once()

	result, err := s.client.PFAdd(context.Background(), "key1", "a", "b", "c")
	s.NoError(err, "There should be no error on PFAdd")
	s.Equal(int64(1), result)

	result, err = s.client.PFAdd(context.Background(), "key2", "d")
	s.NoError(err, "There should be no error on PFAdd")
	s.Equal(int64(1), result)

	result, err = s.client.PFAdd(context.Background(), "key2", "e", "f")
	s.NoError(err, "There should be no error on PFAdd")
	s.Equal(int64(1), result)

	_, err = s.client.PFMerge(context.Background(), "key", "key1", "key2")
	s.NoError(err, "There should be no error on PFMerge")

	result, err = s.client.PFCount(context.Background(), "key")
	s.NoError(err, "There should be no error on PFCount")
	s.Equal(int64(6), result)
}

func TestClientWithMockTestSuite(t *testing.T) {
	suite.Run(t, new(ClientWithMockTestSuite))
}
