//go:build integration

package query_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/db-repo"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/test/suite"
)

type DbRepoQueryTestSuite struct {
	suite.Suite
}

func TestDbRepoQueryTestSuite(t *testing.T) {
	suite.Run(t, new(DbRepoQueryTestSuite))
}

func (s *DbRepoQueryTestSuite) SetupSuite() []suite.Option {
	return []suite.Option{
		suite.WithLogLevel("debug"),
		suite.WithConfigFile("config.test.yml"),
		suite.WithSharedEnvironment(),
	}
}

func (s *DbRepoQueryTestSuite) TestBatchCreate_Success() {
	envConfig := s.Env().Config()
	envLogger := s.Env().Logger()

	repo, err := db_repo.New(envConfig, envLogger, db_repo.Settings{
		Metadata: TestModelMetadata,
	})
	if !s.NoError(err) {
		s.FailNow("there must be no error during repo create")
		return
	}

	models := []*TestModel{
		{
			Name: mdl.Box("nameBatchCreate1"),
		},
		{
			Name: mdl.Box("nameBatchCreate2"),
		},
	}

	err = repo.BatchCreate(s.Env().Context(), models)
	if !s.NoError(err) {
		s.FailNow("there must be no error during create")
		return
	}

	qb := db_repo.NewQueryBuilder()
	qb.Where("name LIKE \"nameBatchCreate%\"")

	results := make([]TestModel, 0)
	err = repo.Query(s.Env().Context(), qb, &results)
	if !s.NoError(err) {
		s.FailNow("there must be no error during query")
		return
	}
	s.Equal(2, len(models), "expected 2 test models")
	s.Equal(*models[0], results[0])
	s.Equal(*models[1], results[1])
}

func (s *DbRepoQueryTestSuite) TestBatchUpdate_Success() {
	envConfig := s.Env().Config()
	envLogger := s.Env().Logger()

	repo, err := db_repo.New(envConfig, envLogger, db_repo.Settings{
		Metadata: TestModelMetadata,
	})
	if !s.NoError(err) {
		s.FailNow("there must be no error during repo create")
		return
	}

	models := []*TestModel{
		{
			Name: mdl.Box("nameBatchUpdate1"),
		},
		{
			Name: mdl.Box("nameBatchUpdate2"),
		},
	}

	err = repo.BatchCreate(s.Env().Context(), models)
	if !s.NoError(err) {
		s.FailNow("there must be no error during create")
		return
	}

	models[0].Name = mdl.Box("nameBatchUpdated1")
	models[1].Name = mdl.Box("nameBatchUpdated2")

	err = repo.BatchUpdate(s.Env().Context(), models)
	if !s.NoError(err) {
		s.FailNow("there must be no error during update")
		return
	}

	qb := db_repo.NewQueryBuilder()
	qb.Where("id in (?, ?)", models[0].Id, models[1].Id)

	results := make([]TestModel, 0)
	err = repo.Query(s.Env().Context(), qb, &results)
	if !s.NoError(err) {
		s.FailNow("there must be no error during query")
		return
	}

	s.Equal(2, len(models), "expected 2 test models")
	s.Equal(models[0].Name, results[0].Name)
	s.Equal(models[1].Name, results[1].Name)
}

func (s *DbRepoQueryTestSuite) TestBatchDelete_Success() {
	envConfig := s.Env().Config()
	envLogger := s.Env().Logger()

	repo, err := db_repo.New(envConfig, envLogger, db_repo.Settings{
		Metadata: TestModelMetadata,
	})
	if !s.NoError(err) {
		s.FailNow("there must be no error during repo create")
		return
	}

	models := []*TestModel{
		{
			Name: mdl.Box("nameBatchDelete1"),
		},
		{
			Name: mdl.Box("nameBatchDelete2"),
		},
	}

	err = repo.BatchCreate(s.Env().Context(), models)
	if !s.NoError(err) {
		s.FailNow("there must be no error during create")
		return
	}

	err = repo.BatchDelete(s.Env().Context(), models)
	if !s.NoError(err) {
		s.FailNow("there must be no error during delete")
		return
	}

	qb := db_repo.NewQueryBuilder()
	qb.Where("id in (?, ?)", models[0].Id, models[1].Id)

	results := make([]TestModel, 0)
	err = repo.Query(s.Env().Context(), qb, &results)
	if !s.NoError(err) {
		s.FailNow("there must be no error during query")
		return
	}

	s.Equal(0, len(results), "expected 0 test models")
}

func (s *DbRepoQueryTestSuite) TestCreate_Success() {
	envConfig := s.Env().Config()
	envLogger := s.Env().Logger()

	repo, err := db_repo.New(envConfig, envLogger, db_repo.Settings{
		Metadata: TestModelMetadata,
	})
	if !s.NoError(err) {
		s.FailNow("there must be no error during repo create")
		return
	}

	model := &TestModel{
		Name: mdl.Box("nameCreate1"),
	}

	err = repo.Create(s.Env().Context(), model)
	if !s.NoError(err) {
		s.FailNow("there must be no error during create")
		return
	}

	where := &TestModel{
		Name: mdl.Box("nameCreate1"),
	}

	qb := db_repo.NewQueryBuilder()
	qb.Where(where)

	models := make([]TestModel, 0)
	err = repo.Query(s.Env().Context(), qb, &models)
	if !s.NoError(err) {
		s.FailNow("there must be no error during query")
		return
	}

	s.Equal(1, len(models), "expected 1 test model")
	s.Equal(*model, models[0])
}

func (s *DbRepoQueryTestSuite) TestRead_Success() {
	envConfig := s.Env().Config()
	envLogger := s.Env().Logger()

	repo, err := db_repo.New(envConfig, envLogger, db_repo.Settings{
		Metadata: TestModelMetadata,
	})
	if !s.NoError(err) {
		s.FailNow("there must be no error during repo create")
		return
	}

	model := &TestModel{
		Name: mdl.Box("nameRead1"),
	}

	err = repo.Create(s.Env().Context(), model)
	if !s.NoError(err) {
		s.FailNow("there must be no error during create")
		return
	}

	readModel := &TestModel{}

	err = repo.Read(s.Env().Context(), model.GetId(), readModel)
	if !s.NoError(err) {
		s.FailNow("there must be no error during query")
		return
	}

	s.Equal(*model, *readModel, "expected db model to match")
}

func (s *DbRepoQueryTestSuite) TestRead_WithPriorCreate() {
	// TODO: do we still need this test?
	envConfig := s.Env().Config()
	envLogger := s.Env().Logger()

	repo, err := db_repo.New(envConfig, envLogger, db_repo.Settings{
		Metadata: TestModelMetadata,
	})
	if !s.NoError(err) {
		s.FailNow("there must be no error during repo create")
		return
	}

	model := &TestModel{
		Name: mdl.Box("nameRead1"),
	}

	err = repo.Create(s.Env().Context(), model)
	if !s.NoError(err) {
		s.FailNow("there must be no error during create")
		return
	}

	model2 := &TestModel{
		Name: mdl.Box("nameRead2"),
	}

	err = repo.Create(s.Env().Context(), model2)
	if !s.NoError(err) {
		s.FailNow("there must be no error during create")
		return
	}

	readModel := &TestModel{}

	err = repo.Read(s.Env().Context(), model.GetId(), readModel)
	if !s.NoError(err) {
		s.FailNow("there must be no error during query")
		return
	}

	s.Equal(*model, *readModel, "expected db model to match")

	readModel2 := &TestModel{}

	err = repo.Read(s.Env().Context(), model2.GetId(), readModel2)
	if !s.NoError(err) {
		s.FailNow("there must be no error during query")
		return
	}

	s.Equal(*model2, *readModel2, "expected db model to match")
}

func (s *DbRepoQueryTestSuite) TestRead_WithAssociations() {
	envConfig := s.Env().Config()
	envLogger := s.Env().Logger()

	repo, err := db_repo.New(envConfig, envLogger, db_repo.Settings{
		Metadata: TestManyMetadata,
	})
	if !s.NoError(err) {
		s.FailNow("there must be no error during repo create")
		return
	}

	other := &TestMany{
		Name: "other",
	}

	err = repo.Create(s.Env().Context(), other)
	if !s.NoError(err) {
		s.FailNow("there must be no error during create")
		return
	}

	many := &TestMany{
		Name: "many",
		Others: []*TestManyToMany{
			{
				Other:   other,
				OtherId: other.Id,
			},
		},
	}

	err = repo.Create(s.Env().Context(), many)
	if !s.NoError(err) {
		s.FailNow("there must be no error during create")
		return
	}

	model := &TestMany{}

	err = repo.Read(s.Env().Context(), many.GetId(), model)
	if !s.NoError(err) {
		s.FailNow("there must be no error during query")
		return
	}

	s.Equal(*many, *model, "expected db model to match")
}

func (s *DbRepoQueryTestSuite) TestUpdate_Success() {
	envConfig := s.Env().Config()
	envLogger := s.Env().Logger()

	repo, err := db_repo.New(envConfig, envLogger, db_repo.Settings{
		Metadata: TestModelMetadata,
	})
	if !s.NoError(err) {
		s.FailNow("there must be no error during repo create")
		return
	}
	model := &TestModel{
		Name: mdl.Box("nameUpdate1"),
	}

	err = repo.Create(s.Env().Context(), model)
	if !s.NoError(err) {
		s.FailNow("there must be no error during create")
		return
	}

	model.Name = mdl.Box("nameUpdate1Updated")

	err = repo.Update(s.Env().Context(), model)
	if !s.NoError(err) {
		s.FailNow("there must be no error during update")
		return
	}

	where := &TestModel{
		Name: mdl.Box("nameUpdate1Updated"),
	}

	qb := db_repo.NewQueryBuilder()
	qb.Where(where)

	models := make([]TestModel, 0)
	err = repo.Query(s.Env().Context(), qb, &models)
	if !s.NoError(err) {
		s.FailNow("there must be no error during query")
		return
	}

	s.Equal(1, len(models), "expected 1 test model")
	s.Equal(*model, models[0])
}

func (s *DbRepoQueryTestSuite) TestDelete_Success() {
	envConfig := s.Env().Config()
	envLogger := s.Env().Logger()

	repo, err := db_repo.New(envConfig, envLogger, db_repo.Settings{
		Metadata: TestModelMetadata,
	})
	if !s.NoError(err) {
		s.FailNow("there must be no error during repo create")
		return
	}

	model := &TestModel{
		Name: mdl.Box("nameDelete1"),
	}

	err = repo.Create(s.Env().Context(), model)
	if !s.NoError(err) {
		s.FailNow("there must be no error during create")
		return
	}

	err = repo.Delete(s.Env().Context(), model)
	if !s.NoError(err) {
		s.FailNow("there must be no error during delete")
		return
	}

	where := &TestModel{
		Name: mdl.Box("nameDelete1"),
	}

	qb := db_repo.NewQueryBuilder()
	qb.Where(where)

	res := make([]TestModel, 0)
	err = repo.Query(s.Env().Context(), qb, &res)
	if !s.NoError(err) {
		s.FailNow("there must be no error during query")
		return
	}

	s.Len(res, 0, "there should be no items found")
}

func (s *DbRepoQueryTestSuite) TestQuery_Success() {
	envConfig := s.Env().Config()
	envLogger := s.Env().Logger()

	repo, err := db_repo.New(envConfig, envLogger, db_repo.Settings{
		Metadata: TestModelMetadata,
	})
	if !s.NoError(err) {
		s.FailNow("there must be no error during repo create")
		return
	}

	model := &TestModel{
		Name: mdl.Box("name1"),
	}

	err = repo.Create(s.Env().Context(), model)
	if !s.NoError(err) {
		s.FailNow("there must be no error during create")
		return
	}

	where := &TestModel{
		Name: mdl.Box("name1"),
	}

	qb := db_repo.NewQueryBuilder()
	qb.Where(where)

	models := make([]TestModel, 0)
	err = repo.Query(s.Env().Context(), qb, &models)
	if !s.NoError(err) {
		s.FailNow("there must be no error during query")
		return
	}

	s.Equal(1, len(models), "expected 1 test model")
	s.Equal(where.Name, models[0].Name)
}
