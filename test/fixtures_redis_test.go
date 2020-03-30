//+build integration

package test_test

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/fixtures"
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/test"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/go-redis/redis"
	"github.com/stretchr/testify/assert"
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
	logger mon.Logger
	mocks  *test.Mocks
}

func (s *FixturesRedisSuite) SetupSuite() {
	setup(s.T())
	mocks, err := test.Boot("test_configs/config.redis.test.yml")

	if err != nil {
		assert.Fail(s.T(), "failed to boot mocks")

		return
	}

	s.mocks = mocks
	s.client = s.mocks.ProvideRedisClient("redis")
	s.logger = mon.NewLogger()
}

func (s *FixturesRedisSuite) TearDownSuite() {
	s.mocks.Shutdown()
}

func TestFixturesRedisSuite(t *testing.T) {
	suite.Run(t, new(FixturesRedisSuite))
}

func (s FixturesRedisSuite) TestRedis() {
	config := cfg.New()
	config.Option(
		cfg.WithConfigFile("test_configs/config.redis.test.yml", "yml"),
		cfg.WithConfigFile("test_configs/config.fixtures_redis.test.yml", "yml"),
		cfg.WithConfigMap(map[string]interface{}{
			"redis_default_addr": fmt.Sprintf("%s:%d", "172.17.0.1", s.mocks.ProvideRedisPort("redis")),
		}),
	)

	// ensure clean start
	_, err := s.client.FlushDB().Result()
	assert.NoError(s.T(), err)

	loader := fixtures.NewFixtureLoader(config, s.logger)

	err = loader.Load(redisDisabledPurgeFixtures())
	assert.NoError(s.T(), err)

	setValue, err := s.client.Get("set_test").Result()

	// should have created the item
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), "bar", setValue)

	rpopValue, err := s.client.LPop("rpush_test").Result()

	// should have created the item
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), "bar", rpopValue)

	rpopValue, err = s.client.LPop("rpush_test").Result()

	// should have created the item
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), "baz", rpopValue)
}

func (s FixturesRedisSuite) TestRedisWithPurge() {
	config := cfg.New()
	config.Option(
		cfg.WithConfigFile("test_configs/config.redis.test.yml", "yml"),
		cfg.WithConfigFile("test_configs/config.fixtures_redis.test.yml", "yml"),
		cfg.WithConfigMap(map[string]interface{}{
			"redis_default_addr": fmt.Sprintf("%s:%d", "172.17.0.1", s.mocks.ProvideRedisPort("redis")),
		}),
	)

	// ensure clean start
	_, err := s.client.FlushDB().Result()
	assert.NoError(s.T(), err)

	loader := fixtures.NewFixtureLoader(config, s.logger)

	err = loader.Load(redisDisabledPurgeFixtures())
	assert.NoError(s.T(), err)

	setValue, err := s.client.Get("set_test").Result()

	// should have created the item
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), "bar", setValue)

	keys, err := s.client.Keys("*").Result()
	assert.NoError(s.T(), err)
	assert.Len(s.T(), keys, 2)

	err = loader.Load(redisEnabledPurgeFixtures())
	assert.NoError(s.T(), err)

	setValue, err = s.client.Get("set_test").Result()

	// should have created the item
	assert.Error(s.T(), err)

	keys, err = s.client.Keys("*").Result()
	assert.NoError(s.T(), err)
	assert.Len(s.T(), keys, 1)

	setValue, err = s.client.Get("set_test_purged").Result()

	// should have created the item
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), "bar", setValue)
}

func (s FixturesRedisSuite) TestRedisKvStore() {
	config := cfg.New()
	config.Option(
		cfg.WithConfigFile("test_configs/config.redis.test.yml", "yml"),
		cfg.WithConfigFile("test_configs/config.fixtures_redis.test.yml", "yml"),
		cfg.WithConfigMap(map[string]interface{}{
			"redis_kvstore_testModel_addr": fmt.Sprintf("%s:%d", "172.17.0.1", s.mocks.ProvideRedisPort("redis")),
		}),
	)

	// ensure clean start
	_, err := s.client.FlushDB().Result()
	assert.NoError(s.T(), err)

	loader := fixtures.NewFixtureLoader(config, s.logger)

	err = loader.Load(redisKvstoreDisabledPurgeFixtures())
	assert.NoError(s.T(), err)

	res, err := s.client.Get("gosoline-integration-test-test-application-kvstore-testModel-kvstore_entry_1").Result()

	// should have created the item
	assert.NoError(s.T(), err)
	assert.JSONEq(s.T(), `{"name":"foo","age":123}`, res)
}

func (s FixturesRedisSuite) TestRedisKvStoreWithPurge() {
	config := cfg.New()
	config.Option(
		cfg.WithConfigFile("test_configs/config.redis.test.yml", "yml"),
		cfg.WithConfigFile("test_configs/config.fixtures_redis.test.yml", "yml"),
		cfg.WithConfigMap(map[string]interface{}{
			"redis_kvstore_testModel_addr": fmt.Sprintf("%s:%d", "172.17.0.1", s.mocks.ProvideRedisPort("redis")),
		}),
	)

	// ensure clean start
	_, err := s.client.FlushDB().Result()
	assert.NoError(s.T(), err)

	loader := fixtures.NewFixtureLoader(config, s.logger)

	err = loader.Load(redisKvstoreDisabledPurgeFixtures())
	assert.NoError(s.T(), err)

	res, err := s.client.Get("gosoline-integration-test-test-application-kvstore-testModel-kvstore_entry_1").Result()

	// should have created the item
	assert.NoError(s.T(), err)
	assert.JSONEq(s.T(), `{"name":"foo","age":123}`, res)

	err = loader.Load(redisKvstoreEnabledPurgeFixtures())
	assert.NoError(s.T(), err)

	res, err = s.client.Get("gosoline-integration-test-test-application-kvstore-testModel-kvstore_entry_1").Result()

	assert.Error(s.T(), err)

	res, err = s.client.Get("gosoline-integration-test-test-application-kvstore-testModel-kvstore_entry_2").Result()

	// should have created the item
	assert.NoError(s.T(), err)
	assert.JSONEq(s.T(), `{"name":"foo","age":123}`, res)
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
