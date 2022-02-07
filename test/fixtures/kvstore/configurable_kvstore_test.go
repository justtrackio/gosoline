//go:build integration && fixtures
// +build integration,fixtures

package kvstore_test

import (
	"context"
	"testing"

	"github.com/justtrackio/gosoline/pkg/fixtures"
	"github.com/justtrackio/gosoline/pkg/kvstore"
	"github.com/justtrackio/gosoline/pkg/test/suite"
)

type KvStoreModel struct {
	Name string `json:"name"`
	Age  uint   `json:"age"`
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
	envContext := s.Env().Context()
	envConfig := s.Env().Config()
	envLogger := s.Env().Logger()

	loader := fixtures.NewFixtureLoader(envContext, envConfig, envLogger)

	err := loader.Load(envContext, buildFixtures())
	s.NoError(err)

	store, err := kvstore.ProvideConfigurableKvStore(envContext, envConfig, envLogger, "test_store")
	s.NoError(err)

	var res KvStoreModel
	found, err := store.Get(context.Background(), "kvstore_entry_1", &res)

	s.NoError(err)
	s.True(found)
	s.Equal(KvStoreModel{
		Name: "foo",
		Age:  12,
	}, res)

	anotherStore, err := kvstore.ProvideConfigurableKvStore(envContext, envConfig, envLogger, "another_test_store")
	s.NoError(err)

	found, err = anotherStore.Get(context.Background(), "kvstore_entry_1", &res)

	s.NoError(err)
	s.True(found)
	s.Equal(KvStoreModel{
		Name: "bar",
		Age:  34,
	}, res)
}

func buildFixtures() []*fixtures.FixtureSet {
	return []*fixtures.FixtureSet{
		{
			Enabled: true,
			Writer:  fixtures.ConfigurableKvStoreFixtureWriterFactory("test_store"),
			Fixtures: []interface{}{
				&fixtures.KvStoreFixture{
					Key: "kvstore_entry_1",
					Value: &KvStoreModel{
						Name: "foo",
						Age:  12,
					},
				},
			},
		},
		{
			Enabled: true,
			Writer:  fixtures.ConfigurableKvStoreFixtureWriterFactory("another_test_store"),
			Fixtures: []interface{}{
				&fixtures.KvStoreFixture{
					Key: "kvstore_entry_1",
					Value: &KvStoreModel{
						Name: "bar",
						Age:  34,
					},
				},
			},
		},
	}
}

func TestConfigurableKvStoreTestSuite(t *testing.T) {
	suite.Run(t, new(ConfigurableKvStoreTestSuite))
}
