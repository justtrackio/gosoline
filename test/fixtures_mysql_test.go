//+build integration

package test_test

import (
	"github.com/applike/gosoline/pkg/cfg"
	db_repo "github.com/applike/gosoline/pkg/db-repo"
	"github.com/applike/gosoline/pkg/fixtures"
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/test"
	gosoAssert "github.com/applike/gosoline/pkg/test/assert"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"testing"
)

type FixturesMysqlSuite struct {
	suite.Suite
	loader fixtures.FixtureLoader
	logger mon.Logger
	mocks  *test.Mocks
}

func (s *FixturesMysqlSuite) SetupSuite() {
	setup(s.T())
	s.mocks = test.Boot("test_configs/config.mysql.test.yml", "test_configs/config.fixtures_mysql.test.yml")

	config := cfg.New()
	config.Option(
		cfg.WithConfigFile("test_configs/config.mysql.test.yml", "yml"),
		cfg.WithConfigFile("test_configs/config.fixtures_mysql.test.yml", "yml"),
	)

	s.logger = mon.NewLogger()
	s.loader = fixtures.NewFixtureLoader(config, s.logger)
}

func (s *FixturesMysqlSuite) TearDownSuite() {
	s.mocks.Shutdown()
}

func TestFixturesMysqlSuite(t *testing.T) {
	suite.Run(t, new(FixturesMysqlSuite))
}

func ormMysqlTestFixtures() []*fixtures.FixtureSet {

	type MysqlTestModel struct {
		db_repo.Model
		Name *string
	}

	var MysqlTestModelMetadata = db_repo.Metadata{
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

	return []*fixtures.FixtureSet{
		{
			Enabled: true,
			Writer:  fixtures.MysqlOrmFixtureWriterFactory(&MysqlTestModelMetadata),
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
			Writer: fixtures.MysqlPlainFixtureWriterFactory(&fixtures.MysqlPlainMetaData{
				TableName: "mysql_plain_writer_test",
				Columns:   []string{"id", "name"},
			}),
			Fixtures: []interface{}{
				fixtures.MysqlPlainFixtureValues{2, "testName2"},
				fixtures.MysqlPlainFixtureValues{2, "testName3"},
			},
		},
	}
}

func (s *FixturesMysqlSuite) TestOrmFixturesMysql() {
	err := s.loader.Load(ormMysqlTestFixtures())
	assert.NoError(s.T(), err)

	db := s.mocks.ProvideMysqlClient("mysql")

	gosoAssert.SqlTableHasOneRowOnly(s.T(), db, "mysql_test_models")
	gosoAssert.SqlColumnHasSpecificValue(s.T(), db, "mysql_test_models", "name", "testName")
}

func (s *FixturesMysqlSuite) TestPlainFixturesMysql() {
	err := s.loader.Load(plainMysqlTestFixtures())
	assert.NoError(s.T(), err)

	db := s.mocks.ProvideMysqlClient("mysql")

	gosoAssert.SqlTableHasOneRowOnly(s.T(), db, "mysql_plain_writer_test")
	gosoAssert.SqlColumnHasSpecificValue(s.T(), db, "mysql_plain_writer_test", "name", "testName3")
}
