//+build integration fixtures

package test_test

import (
	"context"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/fixtures"
	"github.com/applike/gosoline/pkg/kvstore"
	"github.com/applike/gosoline/pkg/log"
	"github.com/stretchr/testify/suite"
	"testing"
)

type ConfigurableKvStoreTestModel struct {
	Name string `json:"name"`
	Age  uint   `json:"age"`
}

type FixturesConfigurableKvStoreSuite struct {
	suite.Suite
	store  kvstore.KvStore
	logger log.Logger
}

func (s *FixturesConfigurableKvStoreSuite) SetupSuite() {
	setup(s.T())
	s.logger = log.NewCliLogger()
	configPath := "test_configs/config.configurable_kvstore.test.yml"

	config := cfg.New()
	_ = config.Option(
		cfg.WithConfigFile(configPath, "yml"),
	)

	var err error
	s.store, err = kvstore.ProvideConfigurableKvStore(config, s.logger, "test_store")
	s.NoError(err)
}

func (s *FixturesConfigurableKvStoreSuite) TearDownSuite() {
}

func TestFixturesConfigurableKvStoreSuite(t *testing.T) {
	suite.Run(t, new(FixturesConfigurableKvStoreSuite))
}

func (s FixturesConfigurableKvStoreSuite) TestConfigurableKvStore() {
	config := cfg.New()
	err := config.Option(
		cfg.WithConfigFile("test_configs/config.configurable_kvstore.test.yml", "yml"),
		cfg.WithConfigFile("test_configs/config.fixtures_configurable_kvstore.test.yml", "yml"),
	)
	s.NoError(err)

	loader := fixtures.NewFixtureLoader(config, s.logger)

	err = loader.Load(configurableKvStoreFixtures())
	s.NoError(err)

	var res ConfigurableKvStoreTestModel
	_, err = s.store.Get(context.Background(), "kvstore_entry_1", &res)

	// should have created the item
	s.NoError(err)

	s.Equal(ConfigurableKvStoreTestModel{
		Name: "foo",
		Age:  123,
	}, res)
}

func configurableKvStoreFixtures() []*fixtures.FixtureSet {
	return []*fixtures.FixtureSet{
		{
			Enabled: true,
			Writer:  fixtures.ConfigurableKvStoreFixtureWriterFactory("test_store"),
			Fixtures: []interface{}{
				&fixtures.KvStoreFixture{
					Key: "kvstore_entry_1",
					Value: &ConfigurableKvStoreTestModel{
						Name: "foo",
						Age:  123,
					},
				},
			},
		},
	}
}
