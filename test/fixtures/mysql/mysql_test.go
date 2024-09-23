//go:build integration && fixtures
// +build integration,fixtures

package mysql_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	gosoAws "github.com/justtrackio/gosoline/pkg/cloud/aws"
	"github.com/justtrackio/gosoline/pkg/db-repo"
	"github.com/justtrackio/gosoline/pkg/fixtures"
	"github.com/justtrackio/gosoline/pkg/mdl"
	gosoAssert "github.com/justtrackio/gosoline/pkg/test/assert"
	"github.com/justtrackio/gosoline/pkg/test/suite"
)

type MysqlTestModel struct {
	db_repo.Model
	Name *string
}

type MysqlTestSuite struct {
	suite.Suite
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
	envContext := s.Env().Context()
	envClient := s.Env().MySql("default").Client()

	loader := s.buildFixtureLoader(envContext)
	fss, err := s.provideMysqlOrmFixtureSets()
	s.NoError(err)

	err = loader.Load(envContext, "default", fss)
	s.NoError(err)

	gosoAssert.SqlTableHasOneRowOnly(s.T(), envClient, "mysql_test_models")
	gosoAssert.SqlColumnHasSpecificValue(s.T(), envClient, "mysql_test_models", "name", "testName")
}

func (s *MysqlTestSuite) TestPlainFixturesMysql() {
	envContext := s.Env().Context()
	envClient := s.Env().MySql("default").Client()

	loader := s.buildFixtureLoader(envContext)
	fss, err := s.provideMysqlPlainFixtureSets()
	s.NoError(err)

	err = loader.Load(envContext, "default", fss)
	s.NoError(err)

	gosoAssert.SqlTableHasOneRowOnly(s.T(), envClient, "mysql_plain_writer_test")
	gosoAssert.SqlColumnHasSpecificValue(s.T(), envClient, "mysql_plain_writer_test", "name", "testName3")
}

func (s *MysqlTestSuite) TestPurgedOrmFixturesMysql() {
	envContext := s.Env().Context()
	envClient := s.Env().MySql("default").Client()

	loader := s.buildFixtureLoader(envContext)
	fss, err := s.provideMysqlOrmFixtureSets()
	s.NoError(err)

	err = loader.Load(envContext, "default", fss)
	s.NoError(err)

	gosoAssert.SqlColumnHasSpecificValue(s.T(), envClient, "mysql_test_models", "name", "testName")

	fss, err = s.provideMysqlOrmFixtureSetsWithPurge()
	s.NoError(err)

	err = loader.Load(envContext, "default", fss)
	s.NoError(err)

	gosoAssert.SqlTableHasOneRowOnly(s.T(), envClient, "mysql_test_models")
	gosoAssert.SqlColumnHasSpecificValue(s.T(), envClient, "mysql_test_models", "name", "purgedBefore")
}

func (s *MysqlTestSuite) TestPurgedPlainFixturesMysql() {
	envContext := s.Env().Context()
	envConfig := s.Env().Config()
	envLogger := s.Env().Logger()
	envClient := s.Env().MySql("default").Client()

	loader := fixtures.NewFixtureLoader(envContext, envConfig, envLogger)
	fss, err := s.provideMysqlPlainFixtureSets()
	s.NoError(err)

	err = loader.Load(envContext, "default", fss)
	s.NoError(err)

	gosoAssert.SqlColumnHasSpecificValue(s.T(), envClient, "mysql_plain_writer_test", "name", "testName3")

	fss, err = s.provideMysqlPlainFixtureSetsWithPurge()
	s.NoError(err)

	err = loader.Load(envContext, "default", fss)
	s.NoError(err)

	gosoAssert.SqlTableHasOneRowOnly(s.T(), envClient, "mysql_plain_writer_test")
	gosoAssert.SqlColumnHasSpecificValue(s.T(), envClient, "mysql_plain_writer_test", "name", "purgedBefore")
}

func (s *MysqlTestSuite) buildFixtureLoader(ctx context.Context) fixtures.FixtureLoader {
	envConfig := s.Env().Config()
	envLogger := s.Env().Logger()

	return fixtures.NewFixtureLoader(ctx, envConfig, envLogger)
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

func (s *MysqlTestSuite) provideMysqlOrmFixtures(data fixtures.NamedFixtures[*MysqlTestModel], purge bool) ([]fixtures.FixtureSet, error) {
	writer, err := fixtures.NewMysqlOrmFixtureWriter(s.Env().Context(), s.Env().Config(), s.Env().Logger(), &MysqlTestModelMetadata)
	if err != nil {
		return nil, fmt.Errorf("failed to create mysql plain fixture writer: %w", err)
	}

	fs := fixtures.NewSimpleFixtureSet(data, writer, fixtures.WithPurge(purge))

	return []fixtures.FixtureSet{
		fs,
	}, nil
}

func (s *MysqlTestSuite) provideMysqlOrmFixtureSets() ([]fixtures.FixtureSet, error) {
	fs := fixtures.NamedFixtures[*MysqlTestModel]{
		&fixtures.NamedFixture[*MysqlTestModel]{
			Name: "test",
			Value: &MysqlTestModel{
				Name: mdl.Box("testName"),
			},
		},
	}

	return s.provideMysqlOrmFixtures(fs, false)
}

func (s *MysqlTestSuite) provideMysqlOrmFixtureSetsWithPurge() ([]fixtures.FixtureSet, error) {
	fs := fixtures.NamedFixtures[*MysqlTestModel]{
		&fixtures.NamedFixture[*MysqlTestModel]{
			Name: "test2",
			Value: &MysqlTestModel{
				Name: mdl.Box("purgedBefore"),
			},
		},
	}

	return s.provideMysqlOrmFixtures(fs, true)
}

func (s *MysqlTestSuite) provideMysqlPlainFixtures(data fixtures.NamedFixtures[fixtures.MysqlPlainFixtureValues], purge bool) ([]fixtures.FixtureSet, error) {
	writer, err := fixtures.NewMysqlPlainFixtureWriter(s.Env().Context(), s.Env().Config(), s.Env().Logger(), &fixtures.MysqlPlainMetaData{
		TableName: "mysql_plain_writer_test",
		Columns:   []string{"id", "name"},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create mysql plain fixture writer: %w", err)
	}

	fs := fixtures.NewSimpleFixtureSet(data, writer, fixtures.WithPurge(purge))

	return []fixtures.FixtureSet{
		fs,
	}, nil
}

func (s *MysqlTestSuite) provideMysqlPlainFixtureSets() ([]fixtures.FixtureSet, error) {
	data := fixtures.NamedFixtures[fixtures.MysqlPlainFixtureValues]{
		&fixtures.NamedFixture[fixtures.MysqlPlainFixtureValues]{
			Name:  "testName2",
			Value: fixtures.MysqlPlainFixtureValues{2, "testName2"},
		},
		&fixtures.NamedFixture[fixtures.MysqlPlainFixtureValues]{
			Name:  "testName3",
			Value: fixtures.MysqlPlainFixtureValues{2, "testName3"},
		},
	}

	return s.provideMysqlPlainFixtures(data, false)
}

func (s *MysqlTestSuite) provideMysqlPlainFixtureSetsWithPurge() ([]fixtures.FixtureSet, error) {
	data := fixtures.NamedFixtures[fixtures.MysqlPlainFixtureValues]{
		&fixtures.NamedFixture[fixtures.MysqlPlainFixtureValues]{
			Name:  "testName4",
			Value: fixtures.MysqlPlainFixtureValues{1, "purgedBefore"},
		},
	}

	return s.provideMysqlPlainFixtures(data, true)
}

func TestMysqlTestSuite(t *testing.T) {
	suite.Run(t, new(MysqlTestSuite))
}
