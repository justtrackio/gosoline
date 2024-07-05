//go:build integration
// +build integration

package query_test

import (
	"context"
	"os"
	"testing"

	gosoAws "github.com/justtrackio/gosoline/pkg/cloud/aws"
	"github.com/justtrackio/gosoline/pkg/db-repo"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/test/suite"
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
}

func (s *DbRepoQueryTestSuite) SetupSuite() []suite.Option {
	err := os.Setenv("AWS_ACCESS_KEY_ID", gosoAws.DefaultAccessKeyID)
	s.NoError(err)

	err = os.Setenv("AWS_SECRET_ACCESS_KEY", gosoAws.DefaultSecretAccessKey)
	s.NoError(err)

	return []suite.Option{
		suite.WithLogLevel("debug"),
		suite.WithConfigFile("config.test.yml"),
		suite.WithSharedEnvironment(),
	}
}

func (s *DbRepoQueryTestSuite) TestCreateCorrectModel() {
	envConfig := s.Env().Config()
	envLogger := s.Env().Logger()

	repo, err := db_repo.New(context.Background(), envConfig, envLogger, db_repo.Settings{
		Metadata: TestModelMetadata,
	})
	s.NoError(err)

	model := &TestModel{
		Name: mdl.Box("nameCreate1"),
	}

	err = repo.Create(context.Background(), model)
	s.NoError(err)

	where := &TestModel{
		Name: mdl.Box("nameCreate1"),
	}

	qb := db_repo.NewQueryBuilder()
	qb.Where(where)

	models := make([]TestModel, 0)
	err = repo.Query(context.Background(), qb, &models)
	s.NoError(err)
	s.Equal(1, len(models), "expected 1 test model")
	s.Equal(*model, models[0])
}

func (s *DbRepoQueryTestSuite) TestCreateWrongModel() {
	envConfig := s.Env().Config()
	envLogger := s.Env().Logger()

	repo, err := db_repo.New(context.Background(), envConfig, envLogger, db_repo.Settings{
		Metadata: TestModelMetadata,
	})
	s.NoError(err)

	model := &WrongTestModel{
		WrongName: mdl.Box("nameCreateWrong1"),
	}

	err = repo.Create(context.Background(), model)
	s.EqualError(err, "cross creating wrong model from repo")
}

func (s *DbRepoQueryTestSuite) TestReadCorrectModel() {
	envConfig := s.Env().Config()
	envLogger := s.Env().Logger()

	repo, err := db_repo.New(context.Background(), envConfig, envLogger, db_repo.Settings{
		Metadata: TestModelMetadata,
	})
	s.NoError(err)

	model := &TestModel{
		Name: mdl.Box("nameRead1"),
	}

	err = repo.Create(context.Background(), model)
	s.NoError(err)

	readModel := &TestModel{}

	err = repo.Read(context.Background(), model.GetId(), readModel)
	s.NoError(err)
	s.Equal(*model, *readModel, "expected db model to match")
}

func (s *DbRepoQueryTestSuite) TestReadWrongModel() {
	envConfig := s.Env().Config()
	envLogger := s.Env().Logger()

	repo, err := db_repo.New(context.Background(), envConfig, envLogger, db_repo.Settings{
		Metadata: TestModelMetadata,
	})
	s.NoError(err)

	model := &WrongTestModel{}

	err = repo.Read(context.Background(), mdl.Box(uint(1)), model)
	s.EqualError(err, "cross reading wrong model from repo")
}

func (s *DbRepoQueryTestSuite) TestUpdateCorrectModel() {
	envConfig := s.Env().Config()
	envLogger := s.Env().Logger()

	repo, err := db_repo.New(context.Background(), envConfig, envLogger, db_repo.Settings{
		Metadata: TestModelMetadata,
	})
	s.NoError(err)

	model := &TestModel{
		Name: mdl.Box("nameUpdate1"),
	}

	err = repo.Create(context.Background(), model)
	s.NoError(err)

	where := &TestModel{
		Name: mdl.Box("nameUpdate1"),
	}

	qb := db_repo.NewQueryBuilder()
	qb.Where(where)

	models := make([]TestModel, 0)
	err = repo.Query(context.Background(), qb, &models)
	s.NoError(err)
	s.Equal(1, len(models), "expected 1 test model")
	s.Equal(*model, models[0])

	model.Name = mdl.Box("nameUpdate1Updated")

	err = repo.Update(context.Background(), model)
	s.NoError(err)

	where = &TestModel{
		Name: mdl.Box("nameUpdate1Updated"),
	}

	qb = db_repo.NewQueryBuilder()
	qb.Where(where)

	models = make([]TestModel, 0)
	err = repo.Query(context.Background(), qb, &models)
	s.NoError(err)
	s.Equal(1, len(models), "expected 1 test model")
	s.Equal(*model, models[0])
}

func (s *DbRepoQueryTestSuite) TestUpdateWrongModel() {
	envConfig := s.Env().Config()
	envLogger := s.Env().Logger()

	repo, err := db_repo.New(context.Background(), envConfig, envLogger, db_repo.Settings{
		Metadata: TestModelMetadata,
	})
	s.NoError(err)

	model := &WrongTestModel{
		WrongName: mdl.Box("nameUpdateWrong1"),
	}

	err = repo.Update(context.Background(), model)
	s.EqualError(err, "cross updating wrong model from repo")
}

func (s *DbRepoQueryTestSuite) TestDeleteCorrectModel() {
	envConfig := s.Env().Config()
	envLogger := s.Env().Logger()

	repo, err := db_repo.New(context.Background(), envConfig, envLogger, db_repo.Settings{
		Metadata: TestModelMetadata,
	})
	s.NoError(err)

	model := &TestModel{
		Name: mdl.Box("nameDelete1"),
	}

	err = repo.Create(context.Background(), model)
	s.NoError(err)

	err = repo.Delete(context.Background(), model)
	s.NoError(err)
}

func (s *DbRepoQueryTestSuite) TestDeleteWrongModel() {
	envConfig := s.Env().Config()
	envLogger := s.Env().Logger()

	repo, err := db_repo.New(context.Background(), envConfig, envLogger, db_repo.Settings{
		Metadata: TestModelMetadata,
	})
	s.NoError(err)

	model := &WrongTestModel{
		WrongName: mdl.Box("nameUpdateWrong1"),
	}

	err = repo.Delete(context.Background(), model)
	s.EqualError(err, "cross deleting wrong model from repo")
}

func (s *DbRepoQueryTestSuite) TestQueryCorrectModel() {
	envConfig := s.Env().Config()
	envLogger := s.Env().Logger()

	repo, err := db_repo.New(context.Background(), envConfig, envLogger, db_repo.Settings{
		Metadata: TestModelMetadata,
	})
	s.NoError(err)

	model := &TestModel{
		Name: mdl.Box("name1"),
	}

	err = repo.Create(context.Background(), model)
	s.NoError(err)

	model = &TestModel{
		Name: mdl.Box("name2"),
	}

	err = repo.Create(context.Background(), model)
	s.NoError(err)

	where := &TestModel{
		Name: mdl.Box("name1"),
	}

	qb := db_repo.NewQueryBuilder()
	qb.Where(where)

	models := make([]TestModel, 0)
	err = repo.Query(context.Background(), qb, &models)
	s.NoError(err)
	s.Equal(1, len(models), "expected 1 test model")
	s.Equal(where.Name, models[0].Name)

	whereStr := "name = ?"

	qb = db_repo.NewQueryBuilder()
	qb.Where(whereStr, mdl.Box("name2"))

	models = make([]TestModel, 0)
	err = repo.Query(context.Background(), qb, &models)
	s.NoError(err)
	s.Equal(1, len(models), "expected 1 test model")
	s.Equal(*model, models[0])
}

func (s *DbRepoQueryTestSuite) TestQueryWrongResultModel() {
	envConfig := s.Env().Config()
	envLogger := s.Env().Logger()

	repo, err := db_repo.New(context.Background(), envConfig, envLogger, db_repo.Settings{
		Metadata: TestModelMetadata,
	})
	s.NoError(err)

	model := &TestModel{
		Name: mdl.Box("name3"),
	}

	err = repo.Create(context.Background(), model)
	s.NoError(err)

	model = &TestModel{
		Name: mdl.Box("name4"),
	}

	err = repo.Create(context.Background(), model)
	s.NoError(err)

	where := &TestModel{
		Name: mdl.Box("name3"),
	}

	qb := db_repo.NewQueryBuilder()
	qb.Where(where)

	models := make([]WrongTestModel, 0)

	err = repo.Query(context.Background(), qb, models)
	s.EqualError(err, "result slice has to be pointer to slice")

	err = repo.Query(context.Background(), qb, &models)
	s.EqualError(err, "cross querying result slice has to be of same model")
}

func (s *DbRepoQueryTestSuite) TestQueryWrongModel() {
	envConfig := s.Env().Config()
	envLogger := s.Env().Logger()

	repo, err := db_repo.New(context.Background(), envConfig, envLogger, db_repo.Settings{
		Metadata: TestModelMetadata,
	})
	s.NoError(err)

	where := &WrongTestModel{
		WrongName: mdl.Box("name1"),
	}

	qb := db_repo.NewQueryBuilder()
	qb.Where(where)

	models := make([]TestModel, 0)
	err = repo.Query(context.Background(), qb, &models)
	s.EqualError(err, "cross querying wrong model from repo")

	whereStruct := WrongTestModel{
		WrongName: mdl.Box("name1"),
	}

	qb = db_repo.NewQueryBuilder()
	qb.Where(whereStruct)

	models = make([]TestModel, 0)
	err = repo.Query(context.Background(), qb, &models)
	s.EqualError(err, "cross querying wrong model from repo")
}

func TestDbRepoQueryTestSuite(t *testing.T) {
	suite.Run(t, new(DbRepoQueryTestSuite))
}
