//+build integration

package test_test

import (
	"context"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/db-repo"
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/test"
	"github.com/stretchr/testify/suite"
	"testing"
)

var TestModelMetadata = db_repo.Metadata{
	ModelId: mdl.ModelId{
		Application: "application",
		Name:        "testModel",
	},
	TableName:  "test_models",
	PrimaryKey: "test_models.id",
	Mappings: db_repo.FieldMappings{
		"testModel.id":   db_repo.NewFieldMapping("test_models.id"),
		"testModel.name": db_repo.NewFieldMapping("test_models.name"),
	},
}

type TestModel struct {
	db_repo.Model
	Name *string
}

type DbRepoQueryTestSuite struct {
	suite.Suite
	logger mon.Logger
	config cfg.Config
	mocks  *test.Mocks
	repo   db_repo.Repository
}

func TestDbRepoQueryTestSuite(t *testing.T) {
	suite.Run(t, new(DbRepoQueryTestSuite))
}

func (s *DbRepoQueryTestSuite) SetupSuite() {
	setup(s.T())
	mocks, err := test.Boot("test_configs/config.mysql.test.yml")

	if !s.NoError(err) {
		return
	}

	s.mocks = mocks

	config := cfg.New()
	_ = config.Option(
		cfg.WithConfigFile("test_configs/config.mysql.test.yml", "yml"),
		cfg.WithConfigFile("test_configs/config.db_repo_query.test.yml", "yml"),
		cfg.WithConfigMap(map[string]interface{}{
			"db.default.uri.port": s.mocks.ProvideMysqlPort("mysql"),
			"db.default.uri.host": s.mocks.ProvideMysqlHost("mysql"),
		}),
	)

	s.config = config
	s.logger = mon.NewLogger()

	s.repo, err = db_repo.New(s.config, s.logger, db_repo.Settings{
		Metadata: TestModelMetadata,
	})

	s.NoError(err)
}

func (s *DbRepoQueryTestSuite) TearDownSuite() {
	s.mocks.Shutdown()
}

func (s *DbRepoQueryTestSuite) TestQueryCorrectModel() {
	ctx := context.Background()

	model := &TestModel{
		Name: mdl.String("name1"),
	}

	err := s.repo.Create(ctx, model)
	s.NoError(err)

	model = &TestModel{
		Name: mdl.String("name2"),
	}

	err = s.repo.Create(ctx, model)
	s.NoError(err)

	where := &TestModel{
		Name: mdl.String("name1"),
	}

	qb := db_repo.NewQueryBuilder()
	qb.Where(where)

	models := make([]TestModel, 0)
	err = s.repo.Query(ctx, qb, &models)
	s.NoError(err)
	s.Equal(1, len(models), "expected 1 test model")

	whereStr := "name = ?"

	qb = db_repo.NewQueryBuilder()
	qb.Where(whereStr, mdl.String("name2"))

	models = make([]TestModel, 0)
	err = s.repo.Query(ctx, qb, &models)
	s.NoError(err)
	s.Equal(1, len(models), "expected 1 test model")
}

func (s *DbRepoQueryTestSuite) TestQueryWrongModel() {
	ctx := context.Background()

	type WrongTestModel struct {
		db_repo.Model
		WrongName *string
	}

	where := &WrongTestModel{
		WrongName: mdl.String("name1"),
	}

	qb := db_repo.NewQueryBuilder()
	qb.Where(where)

	models := make([]TestModel, 0)
	err := s.repo.Query(ctx, qb, &models)
	s.EqualError(err, "cross querying wrong model from repo")

	whereStruct := WrongTestModel{
		WrongName: mdl.String("name1"),
	}

	qb = db_repo.NewQueryBuilder()
	qb.Where(whereStruct)

	models = make([]TestModel, 0)
	err = s.repo.Query(ctx, qb, &models)
	s.EqualError(err, "cross querying wrong model from repo")
}
