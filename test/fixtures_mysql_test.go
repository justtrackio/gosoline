//+build integration fixtures

package test_test

import (
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/db-repo"
	"github.com/applike/gosoline/pkg/fixtures"
	"github.com/applike/gosoline/pkg/log"
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/applike/gosoline/pkg/test"
	gosoAssert "github.com/applike/gosoline/pkg/test/assert"
	"github.com/stretchr/testify/suite"
	"testing"
)

type FixturesMysqlSuite struct {
	suite.Suite
	loader fixtures.FixtureLoader
	logger log.Logger
	mocks  *test.Mocks
}

type MysqlTestModel struct {
	db_repo.Model
	Name *string
}

var MysqlTestModelMetadata = db_repo.Metadata{
	ModelId: mdl.ModelId{
		Name: "test_model",
	},
	TableName:  "mysql_test_models",
	PrimaryKey: "model.id",
	Mappings: db_repo.FieldMappings{
		"test_model.id":   db_repo.NewFieldMapping("test_model.id"),
		"test_model.name": db_repo.NewFieldMapping("test_model.name"),
	},
}

func (s *FixturesMysqlSuite) SetupSuite() {
	setup(s.T())
	mocks, err := test.Boot("test_configs/config.mysql.test.yml", "test_configs/config.fixtures_mysql.test.yml")

	if err != nil {
		s.Fail("failed to boot mocks: %s", err.Error())

		return
	}

	s.mocks = mocks

	config := cfg.New()
	config.Option(
		cfg.WithConfigFile("test_configs/config.mysql.test.yml", "yml"),
		cfg.WithConfigFile("test_configs/config.fixtures_mysql.test.yml", "yml"),
		cfg.WithConfigMap(map[string]interface{}{
			"db.default.uri.port": s.mocks.ProvideMysqlPort("mysql"),
			"db.default.uri.host": s.mocks.ProvideMysqlHost("mysql"),
		}),
	)

	s.logger = log.NewCliLogger()
	s.loader = fixtures.NewFixtureLoader(config, s.logger)
}

func (s *FixturesMysqlSuite) TearDownSuite() {
	s.mocks.Shutdown()
}

func TestFixturesMysqlSuite(t *testing.T) {
	suite.Run(t, new(FixturesMysqlSuite))
}

func ormMysqlTestFixtures() []*fixtures.FixtureSet {
	return []*fixtures.FixtureSet{
		{
			Enabled: true,
			Purge:   false,
			Writer:  fixtures.MysqlOrmFixtureWriterFactory(&MysqlTestModelMetadata),
			Fixtures: []interface{}{
				&MysqlTestModel{
					Name: mdl.String("testName"),
				},
			},
		},
	}
}

func ormMysqlTestFixturesWithPurge() []*fixtures.FixtureSet {
	return []*fixtures.FixtureSet{
		{
			Enabled: true,
			Purge:   true,
			Writer:  fixtures.MysqlOrmFixtureWriterFactory(&MysqlTestModelMetadata),
			Fixtures: []interface{}{
				&MysqlTestModel{
					Name: mdl.String("purgedBefore"),
				},
			},
		},
	}
}

func plainMysqlTestFixtures() []*fixtures.FixtureSet {
	return []*fixtures.FixtureSet{
		{
			Enabled: true,
			Purge:   false,
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

func plainMysqlTestFixturesWithPurge() []*fixtures.FixtureSet {
	return []*fixtures.FixtureSet{
		{
			Enabled: true,
			Purge:   true,
			Writer: fixtures.MysqlPlainFixtureWriterFactory(&fixtures.MysqlPlainMetaData{
				TableName: "mysql_plain_writer_test",
				Columns:   []string{"id", "name"},
			}),
			Fixtures: []interface{}{
				fixtures.MysqlPlainFixtureValues{1, "purgedBefore"},
			},
		},
	}
}

func (s *FixturesMysqlSuite) TestOrmFixturesMysql() {
	err := s.loader.Load(ormMysqlTestFixtures())
	s.NoError(err)

	db := s.mocks.ProvideMysqlClient("mysql")

	gosoAssert.SqlTableHasOneRowOnly(s.T(), db, "mysql_test_models")
	gosoAssert.SqlColumnHasSpecificValue(s.T(), db, "mysql_test_models", "name", "testName")
}

func (s *FixturesMysqlSuite) TestPlainFixturesMysql() {
	err := s.loader.Load(plainMysqlTestFixtures())
	s.NoError(err)

	db := s.mocks.ProvideMysqlClient("mysql")

	gosoAssert.SqlTableHasOneRowOnly(s.T(), db, "mysql_plain_writer_test")
	gosoAssert.SqlColumnHasSpecificValue(s.T(), db, "mysql_plain_writer_test", "name", "testName3")
}

func (s *FixturesMysqlSuite) TestPurgedOrmFixturesMysql() {
	err := s.loader.Load(ormMysqlTestFixtures())
	s.NoError(err)

	db := s.mocks.ProvideMysqlClient("mysql")
	gosoAssert.SqlColumnHasSpecificValue(s.T(), db, "mysql_test_models", "name", "testName")

	err = s.loader.Load(ormMysqlTestFixturesWithPurge())
	s.NoError(err)
	gosoAssert.SqlTableHasOneRowOnly(s.T(), db, "mysql_test_models")
	gosoAssert.SqlColumnHasSpecificValue(s.T(), db, "mysql_test_models", "name", "purgedBefore")
}

func (s *FixturesMysqlSuite) TestPurgedPlainFixturesMysql() {
	err := s.loader.Load(plainMysqlTestFixtures())
	s.NoError(err)

	db := s.mocks.ProvideMysqlClient("mysql")
	gosoAssert.SqlColumnHasSpecificValue(s.T(), db, "mysql_plain_writer_test", "name", "testName3")

	err = s.loader.Load(plainMysqlTestFixturesWithPurge())
	s.NoError(err)
	gosoAssert.SqlTableHasOneRowOnly(s.T(), db, "mysql_plain_writer_test")
	gosoAssert.SqlColumnHasSpecificValue(s.T(), db, "mysql_plain_writer_test", "name", "purgedBefore")
}
