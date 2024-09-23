//go:build integration && fixtures

package migration_golang_migrate

import (
	"fmt"
	"os"
	"testing"

	gosoAws "github.com/justtrackio/gosoline/pkg/cloud/aws"
	"github.com/justtrackio/gosoline/pkg/fixtures"
	gosoAssert "github.com/justtrackio/gosoline/pkg/test/assert"
	"github.com/justtrackio/gosoline/pkg/test/suite"
)

func TestMysqlTestSuite(t *testing.T) {
	suite.Run(t, new(MysqlTestSuite))
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

func (s *MysqlTestSuite) TestPlainFixturesMysql() {
	envContext := s.Env().Context()
	envConfig := s.Env().Config()
	envLogger := s.Env().Logger()
	envClient := s.Env().MySql("default").Client()

	loader := fixtures.NewFixtureLoader(envContext, envConfig, envLogger)

	fss, err := s.provideFixtureSets()
	s.NoError(err)

	err = loader.Load(envContext, "default", fss)
	s.NoError(err)

	gosoAssert.SqlTableHasOneRowOnly(s.T(), envClient, "mysql_plain_writer_test")
	gosoAssert.SqlColumnHasSpecificValue(s.T(), envClient, "mysql_plain_writer_test", "name", "testName3")
}

func (s *MysqlTestSuite) provideFixtureSets() ([]fixtures.FixtureSet, error) {
	writer, err := fixtures.NewMysqlPlainFixtureWriter(s.Env().Context(), s.Env().Config(), s.Env().Logger(), &fixtures.MysqlPlainMetaData{
		TableName: "mysql_plain_writer_test",
		Columns:   []string{"id", "name"},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create mysql plain fixture writer: %w", err)
	}

	dbFixtures := fixtures.NamedFixtures[fixtures.MysqlPlainFixtureValues]{
		&fixtures.NamedFixture[fixtures.MysqlPlainFixtureValues]{
			Name:  "testName2",
			Value: fixtures.MysqlPlainFixtureValues{2, "testName2"},
		},
		&fixtures.NamedFixture[fixtures.MysqlPlainFixtureValues]{
			Name:  "testName2",
			Value: fixtures.MysqlPlainFixtureValues{2, "testName3"},
		},
	}

	return []fixtures.FixtureSet{fixtures.NewSimpleFixtureSet(dbFixtures, writer)}, nil
}
