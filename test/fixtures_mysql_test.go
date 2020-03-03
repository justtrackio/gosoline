//+build integration

package test_test

import (
	"context"
	"github.com/applike/gosoline/pkg/cfg"
	db_repo "github.com/applike/gosoline/pkg/db-repo"
	"github.com/applike/gosoline/pkg/fixtures"
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"testing"
)

type FixturesMysqlSuite struct {
	suite.Suite
	loader fixtures.FixtureLoader
	logger mon.Logger
	repo   db_repo.Repository
}

func (s *FixturesMysqlSuite) SetupSuite() {
	setup(s.T())
	test.Boot("test_configs/config.mysql.test.yml", "test_configs/config.fixtures_mysql.test.yml")

	config := cfg.New()
	config.Option(
		cfg.WithConfigFile("test_configs/config.mysql.test.yml", "yml"),
		cfg.WithConfigFile("test_configs/config.fixtures_mysql.test.yml", "yml"),
	)

	settings := db_repo.Settings{
		AppId:    cfg.GetAppIdFromConfig(config),
		Metadata: TestModelMetadata,
	}

	s.logger = mon.NewLogger()
	s.loader = fixtures.NewFixtureLoader(config, s.logger)
	s.repo = db_repo.New(config, s.logger, settings)
}

func (s *FixturesMysqlSuite) TearDownSuite() {
	test.Shutdown()
}

func TestFixturesMysqlSuite(t *testing.T) {
	suite.Run(t, new(FixturesMysqlSuite))
}

type MysqlTestModel struct {
	db_repo.Model
	Name *string
}

var TestModelMetadata = db_repo.Metadata{
	ModelId: mdl.ModelId{
		Name: "test_model",
	},
	TableName:  "test_models",
	PrimaryKey: "test_model.id",
	Mappings: db_repo.FieldMappings{
		"test_model.id":   db_repo.NewFieldMapping("test_model.id"),
		"test_model.name": db_repo.NewFieldMapping("test_model.name"),
	},
}

func ormMysqlTestFixtures() []*fixtures.FixtureSet {
	return []*fixtures.FixtureSet{
		{
			Enabled: true,
			Writer:  fixtures.MysqlOrmFixtureWriterFactory(&TestModelMetadata),
			Fixtures: []interface{}{
				&MysqlTestModel{
					Name: mdl.String("testName"),
				},
			},
		},
	}
}

func plainMysqlTestFixtures() []*fixtures.FixtureSet {
	return []*fixtures.FixtureSet{
		{
			Enabled: true,
			Writer: fixtures.MysqlPlainFixtureWriterFactory(fixtures.MysqlPlainMetaData{
				TableName: "mysql_test_models",
				Columns:   []string{"id", "name"},
			}),
			Fixtures: []interface{}{
				&fixtures.MysqlPlainFixtureValues{2, "testName2"},
				&fixtures.MysqlPlainFixtureValues{2, "testName3"},
			},
		},
	}
}

func (s *FixturesMysqlSuite) TestOrmFixturesMysql() {
	err := s.loader.Load(ormMysqlTestFixtures())
	assert.NoError(s.T(), err)

	result := MysqlTestModel{}
	_ = s.repo.Read(context.Background(), mdl.Uint(1), &result)

	assert.NoError(s.T(), err)
	if assert.NotNil(s.T(), result.Name) {
		assert.Equal(s.T(), "testName", *result.Name)
	}
}

func (s *FixturesMysqlSuite) TestPlainFixturesMysql() {
	err := s.loader.Load(plainMysqlTestFixtures())
	assert.NoError(s.T(), err)

	result := MysqlTestModel{}
	_ = s.repo.Read(context.Background(), mdl.Uint(2), &result)

	assert.NoError(s.T(), err)
	if assert.NotNil(s.T(), result.Name) {
		assert.Equal(s.T(), "testName3", *result.Name)
	}
}
