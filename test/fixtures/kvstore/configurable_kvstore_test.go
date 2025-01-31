//go:build integration && fixtures

package kvstore_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/fixtures"
	"github.com/justtrackio/gosoline/pkg/kvstore"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/test/suite"
)

type KvStoreModel struct {
	Name string `json:"name"`
	Age  uint   `json:"age"`
}

func TestConfigurableKvStoreTestSuite(t *testing.T) {
	suite.Run(t, new(ConfigurableKvStoreTestSuite))
}

type ConfigurableKvStoreTestSuite struct {
	suite.Suite
}

func (s *ConfigurableKvStoreTestSuite) SetupSuite() []suite.Option {
	return []suite.Option{
		suite.WithLogLevel("debug"),
		suite.WithConfigFile("config.test.yml"),
	}
}

func (s *ConfigurableKvStoreTestSuite) TestConfigurableKvStore() {
	err := s.Env().LoadFixtureSets(s.provideFixtures())
	s.NoError(err)

	envContext := s.Env().Context()
	envConfig := s.Env().Config()
	envLogger := s.Env().Logger()

	store, err := kvstore.ProvideConfigurableKvStore[KvStoreModel](envContext, envConfig, envLogger, "test_store")
	s.NoError(err)

	var res KvStoreModel
	found, err := store.Get(context.Background(), "kvstore_entry_1", &res)

	s.NoError(err)
	s.True(found)
	s.Equal(KvStoreModel{
		Name: "foo",
		Age:  12,
	}, res)

	anotherStore, err := kvstore.ProvideConfigurableKvStore[KvStoreModel](envContext, envConfig, envLogger, "another_test_store")
	s.NoError(err)

	found, err = anotherStore.Get(context.Background(), "kvstore_entry_1", &res)

	s.NoError(err)
	s.True(found)
	s.Equal(KvStoreModel{
		Name: "bar",
		Age:  34,
	}, res)
}

func (s *ConfigurableKvStoreTestSuite) provideFixtureDataTestStore(ctx context.Context, config cfg.Config, logger log.Logger, group string) ([]fixtures.FixtureSet, error) {
	writer, err := kvstore.NewConfigurableKvStoreFixtureWriter[KvStoreModel](s.Env().Context(), s.Env().Config(), s.Env().Logger(), "test_store")
	if err != nil {
		return nil, fmt.Errorf("failed to create kvstore fixture writer: %w", err)
	}

	fs := fixtures.NewSimpleFixtureSet(fixtures.NamedFixtures[*kvstore.KvStoreFixture]{
		&fixtures.NamedFixture[*kvstore.KvStoreFixture]{
			Name: "kvstore_entry_1",
			Value: &kvstore.KvStoreFixture{
				Key: "kvstore_entry_1",
				Value: KvStoreModel{
					Name: "foo",
					Age:  12,
				},
			},
		},
	}, writer)

	return []fixtures.FixtureSet{fs}, nil
}

func (s *ConfigurableKvStoreTestSuite) provideFixtureDataAnotherTestStore(ctx context.Context, config cfg.Config, logger log.Logger, group string) ([]fixtures.FixtureSet, error) {
	writer, err := kvstore.NewConfigurableKvStoreFixtureWriter[KvStoreModel](s.Env().Context(), s.Env().Config(), s.Env().Logger(), "another_test_store")
	if err != nil {
		return nil, fmt.Errorf("failed to create kvstore fixture writer: %w", err)
	}

	fs := fixtures.NewSimpleFixtureSet(fixtures.NamedFixtures[*kvstore.KvStoreFixture]{
		&fixtures.NamedFixture[*kvstore.KvStoreFixture]{
			Name: "kvstore_entry_1",
			Value: &kvstore.KvStoreFixture{
				Key: "kvstore_entry_1",
				Value: KvStoreModel{
					Name: "bar",
					Age:  34,
				},
			},
		},
	}, writer)

	return []fixtures.FixtureSet{fs}, nil
}

func (s *ConfigurableKvStoreTestSuite) provideFixtures() []fixtures.FixtureSetsFactory {
	return []fixtures.FixtureSetsFactory{
		s.provideFixtureDataTestStore,
		s.provideFixtureDataAnotherTestStore,
	}
}
