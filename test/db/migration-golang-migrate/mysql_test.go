//go:build integration && fixtures

package migration_golang_migrate

import (
	"context"
	"fmt"
	"testing"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/db"
	"github.com/justtrackio/gosoline/pkg/fixtures"
	"github.com/justtrackio/gosoline/pkg/log"
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
	return []suite.Option{
		suite.WithLogLevel("debug"),
		suite.WithConfigFile("config.test.yml"),
	}
}

func (s *MysqlTestSuite) TestPlainFixturesMysql() {
	err := s.Env().LoadFixtureSet(s.provideFixtureSets())
	s.NoError(err)

	envClient := s.Env().MySql("default").Client()
	gosoAssert.SqlTableHasOneRowOnly(s.T(), envClient, "mysql_plain_writer_test")
	gosoAssert.SqlColumnHasSpecificValue(s.T(), envClient, "mysql_plain_writer_test", "name", "testName3")
}

func (s *MysqlTestSuite) provideFixtureSets() fixtures.FixtureSetsFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger, group string) ([]fixtures.FixtureSet, error) {
		writer, err := db.NewMysqlPlainFixtureWriter(s.Env().Context(), s.Env().Config(), s.Env().Logger(), &db.MysqlPlainMetaData{
			TableName: "mysql_plain_writer_test",
			Columns:   []string{"id", "name"},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create mysql plain fixture writer: %w", err)
		}

		dbFixtures := fixtures.NamedFixtures[db.MysqlPlainFixtureValues]{
			&fixtures.NamedFixture[db.MysqlPlainFixtureValues]{
				Name:  "testName2",
				Value: db.MysqlPlainFixtureValues{2, "testName2"},
			},
			&fixtures.NamedFixture[db.MysqlPlainFixtureValues]{
				Name:  "testName2",
				Value: db.MysqlPlainFixtureValues{2, "testName3"},
			},
		}

		return []fixtures.FixtureSet{fixtures.NewSimpleFixtureSet(dbFixtures, writer)}, nil
	}
}
