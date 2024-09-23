//go:build integration && fixtures
// +build integration,fixtures

package redis_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/fixtures"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/test/suite"
)

func TestRedisTestSuite(t *testing.T) {
	suite.Run(t, new(RedisTestSuite))
}

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
	envContext := s.Env().Context()
	envConfig := s.Env().Config()
	envLogger := s.Env().Logger()
	ctx := context.Background()

	// ensure clean start
	result, err := redisClient.FlushDB(context.Background()).Result()
	s.NoError(err)
	s.Equal("OK", result)

	loader := fixtures.NewFixtureLoader(envContext, envConfig, envLogger)

	fs1, err := s.provideRedisOpSetFixtureSet()
	s.NoError(err)

	fs2, err := s.provideFixturesOpRpush()
	s.NoError(err)

	err = loader.Load(ctx, "default", []fixtures.FixtureSet{fs1, fs2})
	s.NoError(err)

	// should have created the set_test item
	setValue, err := redisClient.Get(ctx, "set_test").Result()
	s.NoError(err)
	s.Equal("bar", setValue)

	// should have created the rpush_test first item
	rpopValue, err := redisClient.LPop(ctx, "rpush_test").Result()
	s.NoError(err)
	s.Equal("bar", rpopValue)

	// should have created the rpush_test second item
	rpopValue, err = redisClient.LPop(ctx, "rpush_test").Result()
	s.NoError(err)
	s.Equal("baz", rpopValue)
}

func (s *RedisTestSuite) TestRedisWithPurge() {
	redisClient := s.Env().Redis("default").Client()
	envContext := s.Env().Context()
	envConfig := s.Env().Config()
	envLogger := s.Env().Logger()
	ctx := context.Background()

	// ensure clean start
	result, err := redisClient.FlushDB(context.Background()).Result()
	s.NoError(err)
	s.Equal("OK", result)

	loader := fixtures.NewFixtureLoader(envContext, envConfig, envLogger)

	fs1, err := s.provideRedisOpSetFixtureSet()
	s.NoError(err)

	fs2, err := s.provideFixturesOpRpush()
	s.NoError(err)

	err = loader.Load(ctx, "default", []fixtures.FixtureSet{fs1, fs2})
	s.NoError(err)

	// should have created the set_test item
	setValue, err := redisClient.Get(ctx, "set_test").Result()
	s.NoError(err)
	s.Equal("bar", setValue)

	// should have two keys
	keys, err := redisClient.Keys(ctx, "*").Result()
	s.NoError(err)
	s.Len(keys, 2)

	fs, err := s.providePurgeFixtureSet()
	s.NoError(err)

	err = loader.Load(ctx, "default", []fixtures.FixtureSet{fs})
	s.NoError(err)

	// the set_test item should have been purged
	setValue, err = redisClient.Get(ctx, "set_test").Result()
	s.Error(err)
	s.Equal("", setValue)

	// should have only one key
	keys, err = redisClient.Keys(ctx, "*").Result()
	s.NoError(err)
	s.Len(keys, 1)

	// should have created the set_test_purged item
	setValue, err = redisClient.Get(ctx, "set_test_purged").Result()
	s.NoError(err)
	s.Equal("bar", setValue)
}

func (s *RedisTestSuite) TestRedisKvStore() {
	redisClient := s.Env().Redis("default").Client()
	envContext := s.Env().Context()
	envConfig := s.Env().Config()
	envLogger := s.Env().Logger()
	ctx := context.Background()

	// ensure clean start
	result, err := redisClient.FlushDB(context.Background()).Result()
	s.NoError(err)
	s.Equal("OK", result)

	loader := fixtures.NewFixtureLoader(envContext, envConfig, envLogger)

	fss, err := s.provideKvStoreFixtureSet()
	s.NoError(err)

	err = loader.Load(ctx, "default", fss)
	s.NoError(err)

	// should have created the item
	res, err := redisClient.Get(ctx, "gosoline-integration-test-grp-kvstore-testModel-kvstore_entry_1").Result()
	s.NoError(err)
	s.JSONEq(`{"name":"foo","age":123}`, res)
}

func (s *RedisTestSuite) TestRedisKvStoreWithPurge() {
	redisClient := s.Env().Redis("default").Client()
	envContext := s.Env().Context()
	envConfig := s.Env().Config()
	envLogger := s.Env().Logger()
	ctx := context.Background()

	// ensure clean start
	result, err := redisClient.FlushDB(context.Background()).Result()
	s.NoError(err)
	s.Equal("OK", result)

	loader := fixtures.NewFixtureLoader(envContext, envConfig, envLogger)

	fss, err := s.provideKvStoreFixtureSet()
	s.NoError(err)

	err = loader.Load(ctx, "default", fss)
	s.NoError(err)

	// should have created the first item
	res, err := redisClient.Get(ctx, "gosoline-integration-test-grp-kvstore-testModel-kvstore_entry_1").Result()
	s.NoError(err)
	s.JSONEq(`{"name":"foo","age":123}`, res)

	fss, err = s.provideKvStorePurgeFixtureSet()
	s.NoError(err)

	err = loader.Load(ctx, "default", fss)
	s.NoError(err)

	// the first item should have been purged
	res, err = redisClient.Get(ctx, "gosoline-integration-test-grp-kvstore-testModel-kvstore_entry_1").Result()
	s.Error(err)
	s.Equal("", res)

	// should have created the second item
	res, err = redisClient.Get(ctx, "gosoline-integration-test-grp-kvstore-testModel-kvstore_entry_2").Result()
	s.NoError(err)
	s.JSONEq(`{"name":"foo","age":123}`, res)
}

func (s *RedisTestSuite) provideRedisOpSetFixtures(data fixtures.NamedFixtures[*fixtures.RedisFixture], purge bool) (fixtures.FixtureSet, error) {
	redisWriterSet, err := fixtures.NewRedisFixtureWriter(s.Env().Context(), s.Env().Config(), s.Env().Logger(), "default", fixtures.RedisOpSet)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize redis writer: %w", err)
	}

	return fixtures.NewSimpleFixtureSet(data, redisWriterSet, fixtures.WithPurge(purge)), nil
}

func (s *RedisTestSuite) provideRedisOpSetFixtureSet() (fixtures.FixtureSet, error) {
	return s.provideRedisOpSetFixtures(fixtures.NamedFixtures[*fixtures.RedisFixture]{
		&fixtures.NamedFixture[*fixtures.RedisFixture]{
			Name: "redis_set_test",
			Value: &fixtures.RedisFixture{
				Key:    "set_test",
				Value:  "bar",
				Expiry: 1 * time.Hour,
			},
		},
	}, false)
}

func (s *RedisTestSuite) provideFixturesOpRpush() (fixtures.FixtureSet, error) {
	redisWriterRpush, err := fixtures.NewRedisFixtureWriter(s.Env().Context(), s.Env().Config(), s.Env().Logger(), "default", fixtures.RedisOpRpush)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize redis writer: %w", err)
	}

	fs := fixtures.NewSimpleFixtureSet(fixtures.NamedFixtures[*fixtures.RedisFixture]{
		&fixtures.NamedFixture[*fixtures.RedisFixture]{
			Name: "redis_rpush_test",
			Value: &fixtures.RedisFixture{
				Key: "rpush_test",
				Value: []any{
					"bar",
					"baz",
				},
			},
		},
	}, redisWriterRpush)

	return fs, nil
}

func (s *RedisTestSuite) providePurgeFixtureSet() (fixtures.FixtureSet, error) {
	return s.provideRedisOpSetFixtures(fixtures.NamedFixtures[*fixtures.RedisFixture]{
		&fixtures.NamedFixture[*fixtures.RedisFixture]{
			Name: "redis_set_test_purged",
			Value: &fixtures.RedisFixture{
				Key:    "set_test_purged",
				Value:  "bar",
				Expiry: 1 * time.Hour,
			},
		},
	}, true)
}

type Person struct {
	Name string `json:"name"`
	Age  uint   `json:"age"`
}

func (s *RedisTestSuite) provideKvStoreFixtures(data fixtures.NamedFixtures[*fixtures.KvStoreFixture], purge bool) ([]fixtures.FixtureSet, error) {
	kvstoreWriter, err := fixtures.NewRedisKvStoreFixtureWriter[Person](s.Env().Context(), s.Env().Config(), s.Env().Logger(), &mdl.ModelId{
		Project:     "gosoline",
		Environment: "test",
		Family:      "integration-test",
		Group:       "grp",
		Application: "test-application",
		Name:        "testModel",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize kvstore writer: %w", err)
	}

	fs := fixtures.NewSimpleFixtureSet(data, kvstoreWriter, fixtures.WithPurge(purge))

	return []fixtures.FixtureSet{
		fs,
	}, nil
}

func (s *RedisTestSuite) provideKvStoreFixtureSet() ([]fixtures.FixtureSet, error) {
	return s.provideKvStoreFixtures(fixtures.NamedFixtures[*fixtures.KvStoreFixture]{
		&fixtures.NamedFixture[*fixtures.KvStoreFixture]{
			Name: "kvstore_entry_1",
			Value: &fixtures.KvStoreFixture{
				Key: "kvstore_entry_1",
				Value: Person{
					Name: "foo",
					Age:  123,
				},
			},
		},
	}, false)
}

func (s *RedisTestSuite) provideKvStorePurgeFixtureSet() ([]fixtures.FixtureSet, error) {
	return s.provideKvStoreFixtures(fixtures.NamedFixtures[*fixtures.KvStoreFixture]{
		&fixtures.NamedFixture[*fixtures.KvStoreFixture]{
			Name: "kvstore_entry_2",
			Value: &fixtures.KvStoreFixture{
				Key: "kvstore_entry_2",
				Value: Person{
					Name: "foo",
					Age:  123,
				},
			},
		},
	}, true)
}
