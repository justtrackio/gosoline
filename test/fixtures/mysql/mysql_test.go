//go:build integration && fixtures

package mysql_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/db"
	"github.com/justtrackio/gosoline/pkg/db-repo"
	"github.com/justtrackio/gosoline/pkg/fixtures"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
	gosoAssert "github.com/justtrackio/gosoline/pkg/test/assert"
	"github.com/justtrackio/gosoline/pkg/test/suite"
)

type MysqlTestModel struct {
	db_repo.Model
	Name *string
}

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

func (s *MysqlTestSuite) TestOrmFixturesMysql() {
	if err := s.Env().LoadFixtureSet(s.provideMysqlOrmFixtures()); err != nil {
		s.FailNow(err.Error())
	}
	envClient := s.Env().MySql("default").Client()

	gosoAssert.SqlTableHasOneRowOnly(s.T(), envClient, "mysql_test_models")
	gosoAssert.SqlColumnHasSpecificValue(s.T(), envClient, "mysql_test_models", "name", "testName")
}

func (s *MysqlTestSuite) TestPlainFixturesMysql() {
	if err := s.Env().LoadFixtureSet(s.provideMysqlPlainFixtures()); err != nil {
		s.FailNow(err.Error())
	}
	envClient := s.Env().MySql("default").Client()

	gosoAssert.SqlTableHasOneRowOnly(s.T(), envClient, "mysql_plain_writer_test")
	gosoAssert.SqlColumnHasSpecificValue(s.T(), envClient, "mysql_plain_writer_test", "name", "testName3")
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

func (s *MysqlTestSuite) provideMysqlOrmFixtures() fixtures.FixtureSetsFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger, group string) ([]fixtures.FixtureSet, error) {
		writer, err := db_repo.NewMysqlOrmFixtureWriter(s.Env().Context(), s.Env().Config(), s.Env().Logger(), &MysqlTestModelMetadata)
		if err != nil {
			return nil, fmt.Errorf("failed to create mysql plain fixture writer: %w", err)
		}

		data := fixtures.NamedFixtures[*MysqlTestModel]{
			&fixtures.NamedFixture[*MysqlTestModel]{
				Name: "test",
				Value: &MysqlTestModel{
					Name: mdl.Box("testName"),
				},
			},
		}

		fs := fixtures.NewSimpleFixtureSet(data, writer)

		return []fixtures.FixtureSet{
			fs,
		}, nil
	}
}

func (s *MysqlTestSuite) provideMysqlPlainFixtures() fixtures.FixtureSetsFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger, group string) ([]fixtures.FixtureSet, error) {
		writer, err := db.NewMysqlPlainFixtureWriter(s.Env().Context(), s.Env().Config(), s.Env().Logger(), &db.MysqlPlainMetaData{
			TableName: "mysql_plain_writer_test",
			Columns:   []string{"id", "name"},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create mysql plain fixture writer: %w", err)
		}

		data := fixtures.NamedFixtures[db.MysqlPlainFixtureValues]{
			&fixtures.NamedFixture[db.MysqlPlainFixtureValues]{
				Name:  "testName2",
				Value: db.MysqlPlainFixtureValues{2, "testName2"},
			},
			&fixtures.NamedFixture[db.MysqlPlainFixtureValues]{
				Name:  "testName3",
				Value: db.MysqlPlainFixtureValues{2, "testName3"},
			},
		}

		fs := fixtures.NewSimpleFixtureSet(data, writer)

		return []fixtures.FixtureSet{
			fs,
		}, nil
	}
}
