//+build integration

package test_test

import (
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
	s.mocks = test.Boot("test_configs/config.redis.test.yml")
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
	)

	loader := fixtures.NewFixtureLoader(config, s.logger)

	err := loader.Load(redisFixtures())
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

func (s FixturesRedisSuite) TestRedisKvStore() {
	config := cfg.New()
	config.Option(
		cfg.WithConfigFile("test_configs/config.redis.test.yml", "yml"),
		cfg.WithConfigFile("test_configs/config.fixtures_redis.test.yml", "yml"),
	)

	loader := fixtures.NewFixtureLoader(config, s.logger)

	err := loader.Load(redisKvstoreFixtures())
	assert.NoError(s.T(), err)

	res, err := s.client.Get("gosoline-integration-test-test-application-kvstore-testModel-kvstore_entry_1").Result()

	// should have created the item
	assert.NoError(s.T(), err)
	assert.JSONEq(s.T(), `{"name":"foo","age":123}`, res)
}

func redisFixtures() []*fixtures.FixtureSet {
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

func redisKvstoreFixtures() []*fixtures.FixtureSet {
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
