package redis_test

import (
	"errors"
	"github.com/alicebob/miniredis"
	"github.com/applike/gosoline/pkg/mon/mocks"
	"github.com/applike/gosoline/pkg/redis"
	"github.com/elliotchance/redismock"
	baseRedis "github.com/go-redis/redis"
	"github.com/stretchr/testify/suite"
	"testing"
	"time"
)

type ClientTestSuite struct {
	suite.Suite

	server *miniredis.Miniredis
	client redis.Client
}

func (s *ClientTestSuite) SetupTest() {
	server, err := miniredis.Run()

	if err != nil {
		s.FailNow(err.Error(), "can not start miniredis")
		return
	}

	settings := &redis.Settings{}
	logger := mocks.NewLoggerMockedAll()
	metric := mocks.NewMetricWriterMockedAll()

	baseClient := baseRedis.NewClient(&baseRedis.Options{
		Addr: server.Addr(),
	})

	s.server = server
	s.client = redis.NewClientWithInterfaces(logger, baseClient, metric, settings)
}

func (s *ClientTestSuite) TestBLPop() {
	if _, err := s.server.Lpush("list", "value"); err != nil {
		s.FailNow(err.Error(), "can not setup miniredis server")
	}

	res, err := s.client.BLPop(1*time.Second, "list")

	s.NoError(err, "there should be no error on blpop")
	s.Equal("value", res[1])
}

func (s *ClientTestSuite) TestDel() {
	count, err := s.client.Del("test")
	s.NoError(err, "there should be no error on Del")
	s.Equal(0, int(count))

	var ttl time.Duration
	err = s.client.Set("key", "value", ttl)
	s.NoError(err, "there should be no error on Del")

	count, err = s.client.Del("key")
	s.NoError(err, "there should be no error on Del")
	s.Equal(1, int(count))
}

func (s *ClientTestSuite) TestLLen() {
	for i := 0; i < 3; i++ {
		if _, err := s.server.Lpush("list", "value"); err != nil {
			s.FailNow(err.Error(), "can not setup miniredis server")
		}
	}

	res, err := s.client.LLen("list")

	s.NoError(err, "there should be no error on LLen")
	s.Equal(int64(3), res)
}

func (s *ClientTestSuite) TestRPush() {
	count, err := s.client.RPush("list", "v1", "v2", "v3")
	s.NoError(err, "there should be no error on RPush")
	s.Equal(int64(3), count)
}

func (s *ClientTestSuite) TestSet() {
	var ttl time.Duration
	err := s.client.Set("key", "value", ttl)
	s.NoError(err, "there should be no error on Set")

	ttl, _ = time.ParseDuration("1m")
	err = s.client.Set("key", "value", ttl)
	s.NoError(err, "there should be no error on Set with expiration date")
}

func (s *ClientTestSuite) TestHSet() {
	err := s.client.HSet("key", "field", "value")
	s.NoError(err, "there should be no error on HSet")
}

func (s *ClientTestSuite) TestHSetNX() {
	isNewlySet, err := s.client.HSetNX("key", "field", "value")
	s.True(isNewlySet, "the field should be set the first time")
	s.NoError(err, "there should be no error on HSet")

	isNewlySet, err = s.client.HSetNX("key", "field", "value")
	s.False(isNewlySet, "the field should NOT be set the first time")
	s.NoError(err, "there should be no error on HSet")
}

func (s *ClientTestSuite) TestHMSet() {
	err := s.client.HMSet("key", map[string]interface{}{"field": "value"})
	s.NoError(err, "there should be no error on HSet")
}

func (s *ClientTestSuite) TestHMGet() {
	vals, err := s.client.HMGet("key", "field", "value")
	s.NoError(err, "there should be no error on HSet")
	s.Equal([]interface{}{nil, nil}, vals, "there should be no error on HSet")

	err = s.client.HMSet("key", map[string]interface{}{"value": "1"})
	s.NoError(err, "there should be no error on HSet")

	vals, err = s.client.HMGet("key", "field", "value")
	s.NoError(err, "there should be no error on HSet")
	s.Equal([]interface{}{nil, "1"}, vals, "there should be no error on HSet")
}

func (s *ClientTestSuite) TestIncr() {
	val, err := s.client.Incr("key")
	s.NoError(err, "there should be no error on Incr")
	s.Equal(int64(1), val)

	val, err = s.client.Incr("key")
	s.NoError(err, "there should be no error on Incr")
	s.Equal(int64(2), val)

	val, err = s.client.IncrBy("key", int64(3))
	s.NoError(err, "there should be no error on IncrBy")
	s.Equal(int64(5), val)
}

func (s *ClientTestSuite) TestDecr() {
	err := s.client.Set("key", 10, time.Minute*10)

	val, err := s.client.Decr("key")
	s.NoError(err, "there should be no error on Decr")
	s.Equal(int64(9), val)

	val, err = s.client.Decr("key")
	s.NoError(err, "there should be no error on Decr")
	s.Equal(int64(8), val)

	val, err = s.client.DecrBy("key", int64(5))
	s.NoError(err, "there should be no error on DecrBy")
	s.Equal(int64(3), val)
}

func (s *ClientTestSuite) TestExpire() {
	_, _ = s.client.Incr("key")

	result, err := s.client.Expire("key", time.Nanosecond)
	s.NoError(err, "there should be no error on Expire")
	s.True(result)

	amount, err := s.client.Exists("key")
	s.Equal(int64(0), amount)
	s.NoError(err, "there should be no error on Exists")
}

func (s *ClientTestSuite) TestSetWithOOM() {
	var ttl time.Duration

	settings := redis.Settings{}
	writer := mocks.NewMetricWriterMockedAll()
	redisMock := redismock.NewMock()

	redisMock.On("Set").Return(baseRedis.NewStatusResult("", errors.New("OOM command not allowed when used memory > 'maxmemory'"))).Once()
	redisMock.On("Set").Return(baseRedis.NewStatusResult("", nil)).Once()

	logger := mocks.NewLoggerMockedAll()
	client := redis.NewClientWithInterfaces(logger, redisMock, writer, &settings)

	err := client.Set("key", "value", ttl)

	s.NoError(err, "there should be no error on Set with backoff")
	redisMock.AssertExpectations(s.T())
}

func (s *ClientTestSuite) TestSetWithError() {
	var ttl time.Duration

	settings := redis.Settings{}
	writer := mocks.NewMetricWriterMockedAll()
	redisMock := redismock.NewMock()

	redisMock.On("Set").Return(baseRedis.NewStatusResult("", errors.New("random redis error"))).Once()
	redisMock.On("Set").Return(baseRedis.NewStatusResult("", nil)).Times(0)

	logger := mocks.NewLoggerMockedAll()
	client := redis.NewClientWithInterfaces(logger, redisMock, writer, &settings)

	err := client.Set("key", "value", ttl)

	s.NotNil(err, "there should be an error on Set")
	redisMock.AssertExpectations(s.T())
}

func TestClientTestSuite(t *testing.T) {
	suite.Run(t, new(ClientTestSuite))
}
