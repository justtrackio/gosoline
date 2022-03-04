//go:build integration && fixtures
// +build integration,fixtures

package mysql_test

import (
	"context"
	"os"
	"testing"

	gosoAws "github.com/justtrackio/gosoline/pkg/cloud/aws"
	"github.com/justtrackio/gosoline/pkg/db-repo"
	"github.com/justtrackio/gosoline/pkg/fixtures"
	"github.com/justtrackio/gosoline/pkg/mdl"
	gosoAssert "github.com/justtrackio/gosoline/pkg/test/assert"
	"github.com/justtrackio/gosoline/pkg/test/suite"
)

type MysqlTestSuite struct {
	suite.Suite
}

type MysqlTestModel struct {
	db_repo.Model
	Name *string
}

func (s *MysqlTestSuite) SetupSuite() []suite.Option {
	err := os.Setenv("AWS_ACCESS_KEY_ID", gosoAws.DefaultAccessKeyID)
	s.NoError(err)

	err = os.Setenv("AWS_SECRET_ACCESS_KEY", gosoAws.DefaultSecretAccessKey)
	s.NoError(err)

	return []suite.Option{
		suite.WithLogLevel("debug"),
		suite.WithConfigFile("config.test.yml"),
	}
}

func (s *MysqlTestSuite) TestOrmFixturesMysql() {
	envConfig := s.Env().Config()
	envLogger := s.Env().Logger()
	envClient := s.Env().MySql("default").Client()

	loader := fixtures.NewFixtureLoader(context.Background(), envConfig, envLogger)
	err := loader.Load(context.Background(), ormMysqlTestFixtures())
	s.NoError(err)

	gosoAssert.SqlTableHasOneRowOnly(s.T(), envClient, "mysql_test_models")
	gosoAssert.SqlColumnHasSpecificValue(s.T(), envClient, "mysql_test_models", "name", "testName")
}

func (s *MysqlTestSuite) TestPlainFixturesMysql() {
	envConfig := s.Env().Config()
	envLogger := s.Env().Logger()
	envClient := s.Env().MySql("default").Client()

	loader := fixtures.NewFixtureLoader(context.Background(), envConfig, envLogger)
	err := loader.Load(context.Background(), plainMysqlTestFixtures())
	s.NoError(err)

	gosoAssert.SqlTableHasOneRowOnly(s.T(), envClient, "mysql_plain_writer_test")
	gosoAssert.SqlColumnHasSpecificValue(s.T(), envClient, "mysql_plain_writer_test", "name", "testName3")
}

func (s *MysqlTestSuite) TestPurgedOrmFixturesMysql() {
	envConfig := s.Env().Config()
	envLogger := s.Env().Logger()
	envClient := s.Env().MySql("default").Client()

	loader := fixtures.NewFixtureLoader(context.Background(), envConfig, envLogger)
	err := loader.Load(context.Background(), ormMysqlTestFixtures())
	s.NoError(err)

	gosoAssert.SqlColumnHasSpecificValue(s.T(), envClient, "mysql_test_models", "name", "testName")

	err = loader.Load(context.Background(), ormMysqlTestFixturesWithPurge())
	s.NoError(err)

	gosoAssert.SqlTableHasOneRowOnly(s.T(), envClient, "mysql_test_models")
	gosoAssert.SqlColumnHasSpecificValue(s.T(), envClient, "mysql_test_models", "name", "purgedBefore")
}

func (s *MysqlTestSuite) TestPurgedPlainFixturesMysql() {
	envConfig := s.Env().Config()
	envLogger := s.Env().Logger()
	envClient := s.Env().MySql("default").Client()

	loader := fixtures.NewFixtureLoader(context.Background(), envConfig, envLogger)
	err := loader.Load(context.Background(), plainMysqlTestFixtures())
	s.NoError(err)

	gosoAssert.SqlColumnHasSpecificValue(s.T(), envClient, "mysql_plain_writer_test", "name", "testName3")

	err = loader.Load(context.Background(), plainMysqlTestFixturesWithPurge())
	s.NoError(err)

	gosoAssert.SqlTableHasOneRowOnly(s.T(), envClient, "mysql_plain_writer_test")
	gosoAssert.SqlColumnHasSpecificValue(s.T(), envClient, "mysql_plain_writer_test", "name", "purgedBefore")
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

func TestMysqlTestSuite(t *testing.T) {
	suite.Run(t, new(MysqlTestSuite))
}
