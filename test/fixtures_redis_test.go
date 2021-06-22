//+build integration fixtures

package test_test

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/fixtures"
	"github.com/applike/gosoline/pkg/log"
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/applike/gosoline/pkg/test"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/suite"
	"testing"
	"time"
)

type RedisTestModel struct {
	Name string `json:"name"`
	Age  uint   `json:"age"`
}

type FixturesRedisSuite struct {
	suite.Suite
	client *redis.Client
	logger log.Logger
	mocks  *test.Mocks
}

func (s *FixturesRedisSuite) SetupSuite() {
	setup(s.T())
	mocks, err := test.Boot("test_configs/config.redis.test.yml")

	if err != nil {
		s.Fail("failed to boot mocks: %s", err.Error())

		return
	}

	s.mocks = mocks
	s.client = s.mocks.ProvideRedisClient("redis")
	s.logger = log.NewCliLogger()
}

func (s *FixturesRedisSuite) TearDownSuite() {
	s.mocks.Shutdown()
}

func TestFixturesRedisSuite(t *testing.T) {
	suite.Run(t, new(FixturesRedisSuite))
}

func (s FixturesRedisSuite) TestRedis() {
	config := cfg.New()
	_ = config.Option(
		cfg.WithConfigFile("test_configs/config.redis.test.yml", "yml"),
		cfg.WithConfigFile("test_configs/config.fixtures_redis.test.yml", "yml"),
		cfg.WithConfigMap(s.redisConfig()),
	)

	// ensure clean start
	_, err := s.client.FlushDB(context.Background()).Result()
	s.NoError(err)

	loader := fixtures.NewFixtureLoader(config, s.logger)

	err = loader.Load(redisDisabledPurgeFixtures())
	s.NoError(err)

	setValue, err := s.client.Get(context.Background(), "set_test").Result()

	// should have created the item
	s.NoError(err)
	s.Equal("bar", setValue)

	rpopValue, err := s.client.LPop(context.Background(), "rpush_test").Result()

	// should have created the item
	s.NoError(err)
	s.Equal("bar", rpopValue)

	rpopValue, err = s.client.LPop(context.Background(), "rpush_test").Result()

	// should have created the item
	s.NoError(err)
	s.Equal("baz", rpopValue)
}

func (s FixturesRedisSuite) TestRedisWithPurge() {
	config := cfg.New()
	_ = config.Option(
		cfg.WithConfigFile("test_configs/config.redis.test.yml", "yml"),
		cfg.WithConfigFile("test_configs/config.fixtures_redis.test.yml", "yml"),
		cfg.WithConfigMap(s.redisConfig()),
	)

	// ensure clean start
	_, err := s.client.FlushDB(context.Background()).Result()
	s.NoError(err)

	loader := fixtures.NewFixtureLoader(config, s.logger)

	err = loader.Load(redisDisabledPurgeFixtures())
	s.NoError(err)

	setValue, err := s.client.Get(context.Background(), "set_test").Result()

	// should have created the item
	s.NoError(err)
	s.Equal("bar", setValue)

	keys, err := s.client.Keys(context.Background(), "*").Result()
	s.NoError(err)
	s.Len(keys, 2)

	err = loader.Load(redisEnabledPurgeFixtures())
	s.NoError(err)

	setValue, err = s.client.Get(context.Background(), "set_test").Result()

	// should have created the item
	s.Error(err)

	keys, err = s.client.Keys(context.Background(), "*").Result()
	s.NoError(err)
	s.Len(keys, 1)

	setValue, err = s.client.Get(context.Background(), "set_test_purged").Result()

	// should have created the item
	s.NoError(err)
	s.Equal("bar", setValue)
}

func (s FixturesRedisSuite) TestRedisKvStore() {
	config := cfg.New()
	_ = config.Option(
		cfg.WithConfigFile("test_configs/config.redis.test.yml", "yml"),
		cfg.WithConfigFile("test_configs/config.fixtures_redis.test.yml", "yml"),
		cfg.WithConfigMap(s.redisConfig()),
	)

	// ensure clean start
	_, err := s.client.FlushDB(context.Background()).Result()
	s.NoError(err)

	loader := fixtures.NewFixtureLoader(config, s.logger)

	err = loader.Load(redisKvstoreDisabledPurgeFixtures())
	s.NoError(err)

	res, err := s.client.Get(context.Background(), "gosoline-integration-test-test-application-kvstore-testModel-kvstore_entry_1").Result()

	// should have created the item
	s.NoError(err)
	s.JSONEq(`{"name":"foo","age":123}`, res)
}

func (s FixturesRedisSuite) TestRedisKvStoreWithPurge() {
	config := cfg.New()
	_ = config.Option(
		cfg.WithConfigFile("test_configs/config.redis.test.yml", "yml"),
		cfg.WithConfigFile("test_configs/config.fixtures_redis.test.yml", "yml"),
		cfg.WithConfigMap(s.redisConfig()),
	)

	// ensure clean start
	_, err := s.client.FlushDB(context.Background()).Result()
	s.NoError(err)

	loader := fixtures.NewFixtureLoader(config, s.logger)

	err = loader.Load(redisKvstoreDisabledPurgeFixtures())
	s.NoError(err)

	res, err := s.client.Get(context.Background(), "gosoline-integration-test-test-application-kvstore-testModel-kvstore_entry_1").Result()

	// should have created the item
	s.NoError(err)
	s.JSONEq(`{"name":"foo","age":123}`, res)

	err = loader.Load(redisKvstoreEnabledPurgeFixtures())
	s.NoError(err)

	res, err = s.client.Get(context.Background(), "gosoline-integration-test-test-application-kvstore-testModel-kvstore_entry_1").Result()

	s.Error(err)

	res, err = s.client.Get(context.Background(), "gosoline-integration-test-test-application-kvstore-testModel-kvstore_entry_2").Result()

	// should have created the item
	s.NoError(err)
	s.JSONEq(`{"name":"foo","age":123}`, res)
}

func redisDisabledPurgeFixtures() []*fixtures.FixtureSet {
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

func redisEnabledPurgeFixtures() []*fixtures.FixtureSet {
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

func redisKvstoreDisabledPurgeFixtures() []*fixtures.FixtureSet {
	return []*fixtures.FixtureSet{
		{
			Enabled: true,
			Writer: fixtures.RedisKvStoreFixtureWriterFactory(&mdl.ModelId{
				Project:     "gosoline",
				Environment: "test",
				Family:      "integration-test",
				Application: "test-application",
				Name:        "testModel",
			}),
			Fixtures: []interface{}{
				&fixtures.KvStoreFixture{
					Key: "kvstore_entry_1",
					Value: &RedisTestModel{
						Name: "foo",
						Age:  123,
					},
				},
			},
		},
	}
}

func redisKvstoreEnabledPurgeFixtures() []*fixtures.FixtureSet {
	return []*fixtures.FixtureSet{
		{
			Enabled: true,
			Purge:   true,
			Writer: fixtures.RedisKvStoreFixtureWriterFactory(&mdl.ModelId{
				Project:     "gosoline",
				Environment: "test",
				Family:      "integration-test",
				Application: "test-application",
				Name:        "testModel",
			}),
			Fixtures: []interface{}{
				&fixtures.KvStoreFixture{
					Key: "kvstore_entry_2",
					Value: &RedisTestModel{
						Name: "foo",
						Age:  123,
					},
				},
			},
		},
	}
}

func (s FixturesRedisSuite) redisConfig() map[string]interface{} {
	redisAddress := fmt.Sprintf("%s:%d", s.mocks.ProvideRedisHost("redis"), s.mocks.ProvideRedisPort("redis"))
	return map[string]interface{}{
		"redis": map[string]interface{}{
			"default": map[string]interface{}{
				"address": redisAddress,
			},
		},
	}
}
