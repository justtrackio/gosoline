//+build integration

package test_test

import (
	"context"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/db-repo"
	"github.com/applike/gosoline/pkg/log"
	"github.com/applike/gosoline/pkg/mdl"
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

type WrongTestModel struct {
	db_repo.Model
	WrongName *string
}

type DbRepoQueryTestSuite struct {
	suite.Suite
	logger log.Logger
	config cfg.Config
	mocks  *test.Mocks
	repo   db_repo.Repository
}

func TestDbRepoTestSuite(t *testing.T) {
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
	s.logger = log.NewCliLogger()

	s.repo, err = db_repo.New(s.config, s.logger, db_repo.Settings{
		Metadata: TestModelMetadata,
	})

	s.NoError(err)
}

func (s *DbRepoQueryTestSuite) TearDownSuite() {
	s.mocks.Shutdown()
}

func (s *DbRepoQueryTestSuite) TestCreateCorrectModel() {
	ctx := context.Background()

	model := &TestModel{
		Name: mdl.String("nameCreate1"),
	}

	err := s.repo.Create(ctx, model)
	s.NoError(err)

	where := &TestModel{
		Name: mdl.String("nameCreate1"),
	}

	qb := db_repo.NewQueryBuilder()
	qb.Where(where)

	models := make([]TestModel, 0)
	err = s.repo.Query(ctx, qb, &models)
	s.NoError(err)
	s.Equal(1, len(models), "expected 1 test model")
}

func (s *DbRepoQueryTestSuite) TestCreateWrongModel() {
	ctx := context.Background()

	model := &WrongTestModel{
		WrongName: mdl.String("nameCreateWrong1"),
	}

	err := s.repo.Create(ctx, model)
	s.EqualError(err, "cross creating wrong model from repo")
}

func (s *DbRepoQueryTestSuite) TestReadCorrectModel() {
	ctx := context.Background()

	model := &TestModel{
		Name: mdl.String("nameRead1"),
	}

	readModel := &TestModel{}

	err := s.repo.Create(ctx, model)
	s.NoError(err)

	err = s.repo.Read(ctx, model.GetId(), readModel)
	s.NoError(err)
	s.Equal(model.Name, readModel.Name, "expected names to match")
}

func (s *DbRepoQueryTestSuite) TestReadWrongModel() {
	ctx := context.Background()

	model := &WrongTestModel{}

	err := s.repo.Read(ctx, mdl.Uint(1), model)
	s.EqualError(err, "cross reading wrong model from repo")
}

func (s *DbRepoQueryTestSuite) TestUpdateCorrectModel() {
	ctx := context.Background()

	model := &TestModel{
		Name: mdl.String("nameUpdate1"),
	}

	err := s.repo.Create(ctx, model)
	s.NoError(err)

	where := &TestModel{
		Name: mdl.String("nameUpdate1"),
	}

	qb := db_repo.NewQueryBuilder()
	qb.Where(where)

	models := make([]TestModel, 0)
	err = s.repo.Query(ctx, qb, &models)
	s.NoError(err)
	s.Equal(1, len(models), "expected 1 test model")

	model.Name = mdl.String("nameUpdate1Updated")

	err = s.repo.Update(ctx, model)
	s.NoError(err)

	where = &TestModel{
		Name: mdl.String("nameUpdate1Updated"),
	}

	qb = db_repo.NewQueryBuilder()
	qb.Where(where)

	models = make([]TestModel, 0)
	err = s.repo.Query(ctx, qb, &models)
	s.NoError(err)
	s.Equal(1, len(models), "expected 1 test model")
}

func (s *DbRepoQueryTestSuite) TestUpdateWrongModel() {
	ctx := context.Background()

	model := &WrongTestModel{
		WrongName: mdl.String("nameUpdateWrong1"),
	}

	err := s.repo.Update(ctx, model)
	s.EqualError(err, "cross updating wrong model from repo")
}

func (s *DbRepoQueryTestSuite) TestDeleteCorrectModel() {
	ctx := context.Background()

	model := &TestModel{
		Name: mdl.String("nameDelete1"),
	}

	err := s.repo.Create(ctx, model)
	s.NoError(err)

	err = s.repo.Delete(ctx, model)
	s.NoError(err)
}

func (s *DbRepoQueryTestSuite) TestDeleteWrongModel() {
	ctx := context.Background()

	model := &WrongTestModel{
		WrongName: mdl.String("nameUpdateWrong1"),
	}

	err := s.repo.Delete(ctx, model)
	s.EqualError(err, "cross deleting wrong model from repo")
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

func (s *DbRepoQueryTestSuite) TestQueryWrongResultModel() {
	ctx := context.Background()

	model := &TestModel{
		Name: mdl.String("name3"),
	}

	err := s.repo.Create(ctx, model)
	s.NoError(err)

	model = &TestModel{
		Name: mdl.String("name4"),
	}

	err = s.repo.Create(ctx, model)
	s.NoError(err)

	where := &TestModel{
		Name: mdl.String("name3"),
	}

	qb := db_repo.NewQueryBuilder()
	qb.Where(where)

	models := make([]WrongTestModel, 0)

	err = s.repo.Query(ctx, qb, models)
	s.EqualError(err, "result slice has to be pointer to slice")

	err = s.repo.Query(ctx, qb, &models)
	s.EqualError(err, "cross querying result slice has to be of same model")
}

func (s *DbRepoQueryTestSuite) TestQueryWrongModel() {
	ctx := context.Background()

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
