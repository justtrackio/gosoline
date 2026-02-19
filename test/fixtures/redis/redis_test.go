//go:build integration && fixtures

package redis_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/fixtures"
	"github.com/justtrackio/gosoline/pkg/kvstore"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/redis"
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
	// ensure clean start
	redisClient := s.Env().Redis("default").Client()
	ctx := s.T().Context()

	if err := s.Env().LoadFixtureSets([]fixtures.FixtureSetsFactory{
		s.provideRedisOpSetFixtureSet(),
		s.provideFixturesOpRpush(),
	}); err != nil {
		s.FailNow(err.Error())
	}

	// should have created the set_test item
	res := redisClient.Exists(ctx, "set_test")
	s.Equal(int64(1), res.Val(), "key set_test should exist")

	setValue, err := redisClient.Get(ctx, "set_test").Result()
	s.NoError(err)
	s.Equal("bar", setValue, "key set_test should have value 'bar'")

	// should have created the rpush_test first item
	rpopValue, err := redisClient.LPop(ctx, "rpush_test").Result()
	s.NoError(err)
	s.Equal("bar", rpopValue)

	// should have created the rpush_test second item
	rpopValue, err = redisClient.LPop(ctx, "rpush_test").Result()
	s.NoError(err)
	s.Equal("baz", rpopValue)
}

func (s *RedisTestSuite) TestRedisKvStore() {
	// ensure clean start
	redisClient := s.Env().Redis("default").Client()
	ctx := s.T().Context()

	if err := s.Env().LoadFixtureSet(s.provideKvStoreFixtureSet()); err != nil {
		s.FailNow(err.Error())
	}

	cmd := redisClient.Exists(ctx, "prj-test-fam-grp-kvstore-testModel-kvstore_entry_1")
	s.NoError(cmd.Err(), "failed to check existence of key in redis")
	s.Equal(int64(1), cmd.Val(), "key prj-test-fam-grp-kvstore-testModel-kvstore_entry_1 should exist")

	// should have created the item
	res, err := redisClient.Get(ctx, "prj-test-fam-grp-kvstore-testModel-kvstore_entry_1").Result()
	s.NoError(err)
	s.JSONEq(`{"name":"foo","age":123}`, res)
}

func (s *RedisTestSuite) provideRedisOpSetFixtures(data fixtures.NamedFixtures[*redis.RedisFixture]) fixtures.FixtureSetsFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger, group string) ([]fixtures.FixtureSet, error) {
		redisWriterSet, err := redis.NewRedisFixtureWriter(s.Env().Context(), s.Env().Config(), s.Env().Logger(), "default", redis.RedisOpSet)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize redis writer: %w", err)
		}

		return []fixtures.FixtureSet{
			fixtures.NewSimpleFixtureSet(data, redisWriterSet),
		}, nil
	}
}

func (s *RedisTestSuite) provideRedisOpSetFixtureSet() fixtures.FixtureSetsFactory {
	return s.provideRedisOpSetFixtures(fixtures.NamedFixtures[*redis.RedisFixture]{
		&fixtures.NamedFixture[*redis.RedisFixture]{
			Name: "redis_set_test",
			Value: &redis.RedisFixture{
				Key:    "set_test",
				Value:  "bar",
				Expiry: 1 * time.Hour,
			},
		},
	})
}

func (s *RedisTestSuite) provideFixturesOpRpush() fixtures.FixtureSetsFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger, group string) ([]fixtures.FixtureSet, error) {
		redisWriterRpush, err := redis.NewRedisFixtureWriter(s.Env().Context(), s.Env().Config(), s.Env().Logger(), "default", redis.RedisOpRpush)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize redis writer: %w", err)
		}

		fs := fixtures.NewSimpleFixtureSet(fixtures.NamedFixtures[*redis.RedisFixture]{
			&fixtures.NamedFixture[*redis.RedisFixture]{
				Name: "redis_rpush_test",
				Value: &redis.RedisFixture{
					Key: "rpush_test",
					Value: []any{
						"bar",
						"baz",
					},
				},
			},
		}, redisWriterRpush)

		return []fixtures.FixtureSet{fs}, nil
	}
}

type Person struct {
	Name string `json:"name"`
	Age  uint   `json:"age"`
}

func (s *RedisTestSuite) provideKvStoreFixtures(data fixtures.NamedFixtures[*kvstore.KvStoreFixture]) ([]fixtures.FixtureSet, error) {
	kvstoreWriter, err := kvstore.NewRedisKvStoreFixtureWriter[Person](s.Env().Context(), s.Env().Config(), s.Env().Logger(), &mdl.ModelId{
		Name: "testModel",
		Env:  "test",
		App:  "test-application",
		Tags: map[string]string{
			"project": "gosoline",
			"family":  "integration-test",
			"group":   "grp",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize kvstore writer: %w", err)
	}

	fs := fixtures.NewSimpleFixtureSet(data, kvstoreWriter)

	return []fixtures.FixtureSet{
		fs,
	}, nil
}

func (s *RedisTestSuite) provideKvStoreFixtureSet() fixtures.FixtureSetsFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger, group string) ([]fixtures.FixtureSet, error) {
		return s.provideKvStoreFixtures(fixtures.NamedFixtures[*kvstore.KvStoreFixture]{
			&fixtures.NamedFixture[*kvstore.KvStoreFixture]{
				Name: "kvstore_entry_1",
				Value: &kvstore.KvStoreFixture{
					Key: "kvstore_entry_1",
					Value: Person{
						Name: "foo",
						Age:  123,
					},
				},
			},
		})
	}
}
