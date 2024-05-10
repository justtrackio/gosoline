package dbtx_test

import (
	"context"
	"testing"
	"time"

	db_repo "github.com/justtrackio/gosoline/pkg/db-repo"
	"github.com/justtrackio/gosoline/pkg/dbtx"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/test/suite"
)

func TestRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(RepositoryTestSuite))
}

var TestModelMetadata = db_repo.Metadata{
	ModelId: mdl.ModelId{
		Application: "application",
		Name:        "testModel",
	},
	TableName:  "test_entities",
	PrimaryKey: "test_entities.id",
	Mappings: db_repo.FieldMappings{
		"testEntity.id":   db_repo.NewFieldMapping("test_entities.id"),
		"testEntity.name": db_repo.NewFieldMapping("test_entities.name"),
	},
}

var _ dbtx.Entity[string] = &TestEntity{}

type TestEntity struct {
	Id        string
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (t *TestEntity) SetCreatedAt(createdAt time.Time) {
	t.CreatedAt = createdAt
}

func (t *TestEntity) SetUpdatedAt(updatedAt time.Time) {
	t.UpdatedAt = updatedAt
}

func (t *TestEntity) GetId() string {
	return t.Id
}

func (t *TestEntity) GetCreatedAt() time.Time {
	return t.CreatedAt
}

func (t *TestEntity) GetUpdatedAt() time.Time {
	return t.UpdatedAt
}

type RepositoryTestSuite struct {
	suite.Suite
	repo dbtx.Repository[string, *TestEntity]
}

func (s *RepositoryTestSuite) SetupSuite() []suite.Option {
	return []suite.Option{
		suite.WithLogLevel("debug"),
		suite.WithConfigFile("config.test.yml"),
		suite.WithSharedEnvironment(),
		suite.WithEnvSetup(s.setupEnv),
	}
}

func (s *RepositoryTestSuite) setupEnv() (err error) {
	envConfig := s.Env().Config()
	envLogger := s.Env().Logger()

	s.repo, err = dbtx.New[string, *TestEntity](envConfig, envLogger, db_repo.Settings{
		Metadata: TestModelMetadata,
	})

	return
}

func (s *RepositoryTestSuite) TestCreate() {
	ent := &TestEntity{
		Id:   "foo",
		Name: "bar",
	}

	tx := s.repo.NewTx(context.Background())
	err := s.repo.Create(tx, ent)
	s.NoError(err)

	err = tx.Commit()
	s.NoError(err)
}
