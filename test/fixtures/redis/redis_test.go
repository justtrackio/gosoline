//go:build integration && fixtures
// +build integration,fixtures

package redis_test

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/justtrackio/gosoline/pkg/fixtures"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/test/suite"
)

type RedisTestSuite struct {
	suite.Suite
}

func (s *RedisTestSuite) SetupSuite() []suite.Option {
	return []suite.Option{
		suite.WithLogLevel("debug"),
		suite.WithConfigFile("config.test.yml"),
	}
}

func (s *RedisTestSuite) TestRedis() {
	redisClient := s.Env().Redis("default").Client()
	envConfig := s.Env().Config()
	envLogger := s.Env().Logger()

	// ensure clean start
	result, err := redisClient.FlushDB(context.Background()).Result()
	s.NoError(err)
	s.Equal("OK", result)

	loader := fixtures.NewFixtureLoader(context.Background(), envConfig, envLogger)

	err = loader.Load(context.Background(), disabledPurgeFixtures())
	s.NoError(err)

	// should have created the set_test item
	setValue, err := redisClient.Get(context.Background(), "set_test").Result()
	s.NoError(err)
	s.Equal("bar", setValue)

	// should have created the rpush_test first item
	rpopValue, err := redisClient.LPop(context.Background(), "rpush_test").Result()
	s.NoError(err)
	s.Equal("bar", rpopValue)

	// should have created the rpush_test second item
	rpopValue, err = redisClient.LPop(context.Background(), "rpush_test").Result()
	s.NoError(err)
	s.Equal("baz", rpopValue)
}

func (s *RedisTestSuite) TestRedisWithPurge() {
	redisClient := s.Env().Redis("default").Client()
	envConfig := s.Env().Config()
	envLogger := s.Env().Logger()

	// ensure clean start
	result, err := redisClient.FlushDB(context.Background()).Result()
	s.NoError(err)
	s.Equal("OK", result)

	loader := fixtures.NewFixtureLoader(context.Background(), envConfig, envLogger)

	err = loader.Load(context.Background(), disabledPurgeFixtures())
	s.NoError(err)

	// should have created the set_test item
	setValue, err := redisClient.Get(context.Background(), "set_test").Result()
	s.NoError(err)
	s.Equal("bar", setValue)

	// should have two keys
	keys, err := redisClient.Keys(context.Background(), "*").Result()
	s.NoError(err)
	s.Len(keys, 2)

	err = loader.Load(context.Background(), enabledPurgeFixtures())
	s.NoError(err)

	// the set_test item should have been purged
	setValue, err = redisClient.Get(context.Background(), "set_test").Result()
	s.Error(err)

	// should have only one key
	keys, err = redisClient.Keys(context.Background(), "*").Result()
	s.NoError(err)
	s.Len(keys, 1)

	// should have created the set_test_purged item
	setValue, err = redisClient.Get(context.Background(), "set_test_purged").Result()
	s.NoError(err)
	s.Equal("bar", setValue)
}

func (s *RedisTestSuite) TestRedisKvStore() {
	redisClient := s.Env().Redis("default").Client()
	envConfig := s.Env().Config()
	envLogger := s.Env().Logger()

	// ensure clean start
	result, err := redisClient.FlushDB(context.Background()).Result()
	s.NoError(err)
	s.Equal("OK", result)

	loader := fixtures.NewFixtureLoader(context.Background(), envConfig, envLogger)

	err = loader.Load(context.Background(), kvStoreDisabledPurgeFixtures())
	s.NoError(err)

	// should have created the item
	res, err := redisClient.Get(context.Background(), "gosoline-integration-test-grp-kvstore-testModel-kvstore_entry_1").Result()
	s.NoError(err)
	s.JSONEq(`{"name":"foo","age":123}`, res)
}

func (s *RedisTestSuite) TestRedisKvStoreWithPurge() {
	redisClient := s.Env().Redis("default").Client()
	envConfig := s.Env().Config()
	envLogger := s.Env().Logger()

	// ensure clean start
	result, err := redisClient.FlushDB(context.Background()).Result()
	s.NoError(err)
	s.Equal("OK", result)

	loader := fixtures.NewFixtureLoader(context.Background(), envConfig, envLogger)

	err = loader.Load(context.Background(), kvStoreDisabledPurgeFixtures())
	s.NoError(err)

	// should have created the first item
	res, err := redisClient.Get(context.Background(), "gosoline-integration-test-grp-kvstore-testModel-kvstore_entry_1").Result()
	s.NoError(err)
	s.JSONEq(`{"name":"foo","age":123}`, res)

	err = loader.Load(context.Background(), kvStoreEnabledPurgeFixtures())
	s.NoError(err)

	// the first item should have been purged
	res, err = redisClient.Get(context.Background(), "gosoline-integration-test-grp-kvstore-testModel-kvstore_entry_1").Result()
	s.Error(err)

	// should have created the second item
	res, err = redisClient.Get(context.Background(), "gosoline-integration-test-grp-kvstore-testModel-kvstore_entry_2").Result()
	s.NoError(err)
	s.JSONEq(`{"name":"foo","age":123}`, res)
}

func disabledPurgeFixtures() []*fixtures.FixtureSet {
	return []*fixtures.FixtureSet{
		{
			Enabled: true,
			Writer:  fixtures.RedisFixtureWriterFactory(aws.String("default"), aws.String(fixtures.RedisOpSet)),
			Fixtures: []interface{}{
				&fixtures.RedisFixture{
					Key:    "set_test",
					Value:  "bar",
					Expiry: 1 * time.Hour,
				},
			},
		},
		{
			Enabled: true,
			Writer:  fixtures.RedisFixtureWriterFactory(aws.String("default"), aws.String(fixtures.RedisOpRpush)),
			Fixtures: []interface{}{
				&fixtures.RedisFixture{
					Key: "rpush_test",
					Value: []interface{}{
						"bar",
						"baz",
					},
				},
			},
		},
	}
}

func enabledPurgeFixtures() []*fixtures.FixtureSet {
	return []*fixtures.FixtureSet{
		{
			Enabled: true,
			Purge:   true,
			Writer:  fixtures.RedisFixtureWriterFactory(aws.String("default"), aws.String(fixtures.RedisOpSet)),
			Fixtures: []interface{}{
				&fixtures.RedisFixture{
					Key:    "set_test_purged",
					Value:  "bar",
					Expiry: 1 * time.Hour,
				},
			},
		},
	}
}

type Person struct {
	Name string `json:"name"`
	Age  uint   `json:"age"`
}

func kvStoreDisabledPurgeFixtures() []*fixtures.FixtureSet {
	return []*fixtures.FixtureSet{
		{
			Enabled: true,
			Writer: fixtures.RedisKvStoreFixtureWriterFactory[Person](&mdl.ModelId{
				Project:     "gosoline",
				Environment: "test",
				Family:      "integration-test",
				Group:       "grp",
				Application: "test-application",
				Name:        "testModel",
			}),
			Fixtures: []interface{}{
				&fixtures.KvStoreFixture{
					Key: "kvstore_entry_1",
					Value: Person{
						Name: "foo",
						Age:  123,
					},
				},
			},
		},
	}
}

func kvStoreEnabledPurgeFixtures() []*fixtures.FixtureSet {
	return []*fixtures.FixtureSet{
		{
			Enabled: true,
			Purge:   true,
			Writer: fixtures.RedisKvStoreFixtureWriterFactory[Person](&mdl.ModelId{
				Project:     "gosoline",
				Environment: "test",
				Family:      "integration-test",
				Group:       "grp",
				Application: "test-application",
				Name:        "testModel",
			}),
			Fixtures: []interface{}{
				&fixtures.KvStoreFixture{
					Key: "kvstore_entry_2",
					Value: Person{
						Name: "foo",
						Age:  123,
					},
				},
			},
		},
	}
}

func TestRedisTestSuite(t *testing.T) {
	suite.Run(t, new(RedisTestSuite))
}
