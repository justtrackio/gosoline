package redis_test

import (
	"testing"
	"time"

	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/redis"
	redisMocks "github.com/justtrackio/gosoline/pkg/redis/mocks"
	"github.com/justtrackio/gosoline/pkg/test/matcher"
	"github.com/stretchr/testify/suite"
)

func TestRedisFixtureWriterTestSuite(t *testing.T) {
	suite.Run(t, new(RedisFixtureWriterTestSuite))
}

type RedisFixtureWriterTestSuite struct {
	suite.Suite
}

func (s *RedisFixtureWriterTestSuite) TestWriteSetEmpty() {
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(s.T()))
	client := redisMocks.NewClient(s.T())

	writer := redis.NewRedisFixtureWriterWithInterfaces(logger, client, redis.RedisOpSet)

	err := writer.Write(s.T().Context(), []any{})
	s.NoError(err)
}

func (s *RedisFixtureWriterTestSuite) TestWriteSetWithoutTTL() {
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(s.T()))
	client := redisMocks.NewClient(s.T())

	fixtures := []any{
		&redis.RedisFixture{Key: "key1", Value: "value1", Expiry: 0},
		&redis.RedisFixture{Key: "key2", Value: "value2", Expiry: 0},
	}

	// MSet should be called with all key-value pairs
	client.EXPECT().MSet(matcher.Context, "key1", "value1", "key2", "value2").Return(nil)

	writer := redis.NewRedisFixtureWriterWithInterfaces(logger, client, redis.RedisOpSet)

	err := writer.Write(s.T().Context(), fixtures)
	s.NoError(err)
}

func (s *RedisFixtureWriterTestSuite) TestWriteSetWithTTL() {
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(s.T()))
	client := redisMocks.NewClient(s.T())

	fixtures := []any{
		&redis.RedisFixture{Key: "key1", Value: "value1", Expiry: 5 * time.Minute},
		&redis.RedisFixture{Key: "key2", Value: "value2", Expiry: 10 * time.Minute},
	}

	// Individual Set calls should be made for fixtures with TTL
	client.EXPECT().Set(matcher.Context, "key1", "value1", 5*time.Minute).Return(nil)
	client.EXPECT().Set(matcher.Context, "key2", "value2", 10*time.Minute).Return(nil)

	writer := redis.NewRedisFixtureWriterWithInterfaces(logger, client, redis.RedisOpSet)

	err := writer.Write(s.T().Context(), fixtures)
	s.NoError(err)
}

func (s *RedisFixtureWriterTestSuite) TestWriteSetMixedTTL() {
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(s.T()))
	client := redisMocks.NewClient(s.T())

	fixtures := []any{
		&redis.RedisFixture{Key: "key1", Value: "value1", Expiry: 0},               // No TTL - use MSet
		&redis.RedisFixture{Key: "key2", Value: "value2", Expiry: 5 * time.Minute}, // With TTL - use Set
		&redis.RedisFixture{Key: "key3", Value: "value3", Expiry: 0},               // No TTL - use MSet
	}

	// MSet should be called with all no-TTL key-value pairs
	client.EXPECT().MSet(matcher.Context, "key1", "value1", "key3", "value3").Return(nil)

	// Individual Set call for fixture with TTL
	client.EXPECT().Set(matcher.Context, "key2", "value2", 5*time.Minute).Return(nil)

	writer := redis.NewRedisFixtureWriterWithInterfaces(logger, client, redis.RedisOpSet)

	err := writer.Write(s.T().Context(), fixtures)
	s.NoError(err)
}

func (s *RedisFixtureWriterTestSuite) TestWriteRpushEmpty() {
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(s.T()))
	client := redisMocks.NewClient(s.T())

	writer := redis.NewRedisFixtureWriterWithInterfaces(logger, client, redis.RedisOpRpush)

	err := writer.Write(s.T().Context(), []any{})
	s.NoError(err)
}

func (s *RedisFixtureWriterTestSuite) TestWriteRpush() {
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(s.T()))
	client := redisMocks.NewClient(s.T())

	fixtures := []any{
		&redis.RedisFixture{Key: "list1", Value: []any{"a", "b", "c"}},
		&redis.RedisFixture{Key: "list2", Value: []any{"x", "y"}},
	}

	// RPUSH operates on different keys, so individual calls are made
	client.EXPECT().RPush(matcher.Context, "list1", "a", "b", "c").Return(int64(3), nil)
	client.EXPECT().RPush(matcher.Context, "list2", "x", "y").Return(int64(2), nil)

	writer := redis.NewRedisFixtureWriterWithInterfaces(logger, client, redis.RedisOpRpush)

	err := writer.Write(s.T().Context(), fixtures)
	s.NoError(err)
}

func (s *RedisFixtureWriterTestSuite) TestWriteUnknownOperation() {
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(s.T()))
	client := redisMocks.NewClient(s.T())

	fixtures := []any{
		&redis.RedisFixture{Key: "key1", Value: "value1"},
	}

	writer := redis.NewRedisFixtureWriterWithInterfaces(logger, client, "UNKNOWN")

	err := writer.Write(s.T().Context(), fixtures)
	s.Error(err)
	s.Contains(err.Error(), "no handler for operation: UNKNOWN")
}
