//go:build integration

package change_history_test

import (
	"context"
	"os"
	"testing"

	gosoAws "github.com/justtrackio/gosoline/pkg/cloud/aws"
	"github.com/justtrackio/gosoline/pkg/db-repo"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/test/suite"
)

type TestModel1 struct {
	db_repo.Model
	Name *string
}

type TestModel1HistoryEntry struct {
	db_repo.ChangeHistoryModel
	TestModel1
}

var TestModel1Metadata = db_repo.Metadata{
	ModelId: mdl.ModelId{
		Application: "application",
		Name:        "testModel1",
	},
	TableName:  "test_model1",
	PrimaryKey: "test_model1.id",
	Mappings: db_repo.FieldMappings{
		"testModel1.id":   db_repo.NewFieldMapping("test_model1.id"),
		"testModel1.name": db_repo.NewFieldMapping("test_model1.name"),
	},
}

var TestHistoryModel1Metadata = db_repo.Metadata{
	ModelId: mdl.ModelId{
		Application: "application",
		Name:        "testModel1HistoryEntry",
	},
	TableName:  "test_model1_history_entries",
	PrimaryKey: "test_model1_history_entries.id",
	Mappings: db_repo.FieldMappings{
		"testModel1HistoryEntry.id":   db_repo.NewFieldMapping("test_model1_history_entries.id"),
		"testModel1HistoryEntry.name": db_repo.NewFieldMapping("test_model1_history_entries.name"),
	},
}

type TestModel2 struct {
	db_repo.Model
	Name         *string
	Foo          *string
	ChangeAuthor *string
}

type TestModel2HistoryEntry struct {
	db_repo.ChangeHistoryModel
	TestModel2
}

var TestModel2Metadata = db_repo.Metadata{
	ModelId: mdl.ModelId{
		Application: "application",
		Name:        "testModel2",
	},
	TableName:  "test_model2",
	PrimaryKey: "test_model2.id",
	Mappings: db_repo.FieldMappings{
		"testModel2.id":           db_repo.NewFieldMapping("test_model2.id"),
		"testModel2.name":         db_repo.NewFieldMapping("test_model2.name"),
		"testModel2.foo":          db_repo.NewFieldMapping("test_model2.foo"),
		"testModel2.changeAuthor": db_repo.NewFieldMapping("test_model2.change_author"),
	},
}

var TestHistoryModel2Metadata = db_repo.Metadata{
	ModelId: mdl.ModelId{
		Application: "application",
		Name:        "testModel2HistoryEntry",
	},
	TableName:  "test_model2_history_entries",
	PrimaryKey: "test_model2_history_entries.id",
	Mappings: db_repo.FieldMappings{
		"testModel2HistoryEntry.id":           db_repo.NewFieldMapping("test_model2_history_entries.id"),
		"testModel2HistoryEntry.name":         db_repo.NewFieldMapping("test_model2_history_entries.name"),
		"testModel2HistoryEntry.foo":          db_repo.NewFieldMapping("test_model2_history_entries.foo"),
		"testModel2HistoryEntry.changeAuthor": db_repo.NewFieldMapping("test_model2_history_entries.change_author"),
	},
}

type ChangeHistoryTestSuite struct {
	suite.Suite
}

func (s *ChangeHistoryTestSuite) SetupSuite() []suite.Option {
	err := os.Setenv("AWS_ACCESS_KEY_ID", gosoAws.DefaultAccessKeyID)
	s.NoError(err)

	err = os.Setenv("AWS_SECRET_ACCESS_KEY", gosoAws.DefaultSecretAccessKey)
	s.NoError(err)

	return []suite.Option{
		suite.WithLogLevel("debug"),
		suite.WithConfigFile("config.test.yml"),
	}
}

func (s *ChangeHistoryTestSuite) TestChangeHistoryMigration_Migrate_CreateTable() {
	envConfig := s.Env().Config()
	envLogger := s.Env().Logger()

	modelRepo, err := db_repo.New(envConfig, envLogger, db_repo.Settings{
		Metadata: TestModel1Metadata,
	})
	s.NoError(err)

	modelHistoryRepo, err := db_repo.New(envConfig, envLogger, db_repo.Settings{
		Metadata: TestHistoryModel1Metadata,
	})
	s.NoError(err)

	historyManager, err := db_repo.NewChangeHistoryManager(envConfig, envLogger)
	s.NoError(err)

	err = historyManager.RunMigration(&TestModel1{})
	if !s.NoError(err) {
		s.FailNow("there must be no error running the history migration")
		return
	}

	model := &TestModel1{
		Name: mdl.Box("name1"),
	}

	err = modelRepo.Create(context.Background(), model)
	s.NoError(err)

	model.Name = mdl.Box("name2")
	err = modelRepo.Update(context.Background(), model)
	s.NoError(err)

	err = modelRepo.Delete(context.Background(), model)
	s.NoError(err)

	entries := make([]*TestModel1HistoryEntry, 0)
	err = modelHistoryRepo.Query(context.Background(), &db_repo.QueryBuilder{}, &entries)
	s.NoError(err)
	s.Equal(3, len(entries), "expected 3 change history entries")

	s.Equal(1, entries[0].ChangeHistoryRevision)
	s.Equal("insert", entries[0].ChangeHistoryAction)
	s.Equal("name1", *entries[0].Name)

	s.Equal(2, entries[1].ChangeHistoryRevision)
	s.Equal("update", entries[1].ChangeHistoryAction)
	s.Equal("name2", *entries[1].Name)

	s.Equal(3, entries[2].ChangeHistoryRevision)
	s.Equal("delete", entries[2].ChangeHistoryAction)
	s.Equal("name2", *entries[2].Name)
}

func (s *ChangeHistoryTestSuite) TestChangeHistoryMigration_Migrate_UpdateTable() {
	envConfig := s.Env().Config()
	envLogger := s.Env().Logger()

	modelRepo, err := db_repo.New(envConfig, envLogger, db_repo.Settings{
		Metadata: TestModel2Metadata,
	})
	s.NoError(err)

	modelHistoryRepo, err := db_repo.New(envConfig, envLogger, db_repo.Settings{
		Metadata: TestHistoryModel2Metadata,
	})
	s.NoError(err)

	historyManager, err := db_repo.NewChangeHistoryManager(envConfig, envLogger)
	s.NoError(err)

	err = historyManager.RunMigration(&TestModel2{})
	s.NoError(err)

	model := &TestModel2{
		Name:         mdl.Box("name1"),
		Foo:          mdl.Box("foo1"),
		ChangeAuthor: mdl.Box("john@example.com"),
	}

	err = modelRepo.Create(context.Background(), model)
	s.NoError(err)

	model.Foo = mdl.Box("foo2")
	err = modelRepo.Update(context.Background(), model)
	s.NoError(err)

	err = modelRepo.Delete(context.Background(), model)
	s.NoError(err)

	entries := make([]*TestModel2HistoryEntry, 0)
	err = modelHistoryRepo.Query(context.Background(), &db_repo.QueryBuilder{}, &entries)
	s.NoError(err)
	s.Equal(3, len(entries), "expected 3 change history entries")

	s.Equal(1, entries[0].ChangeHistoryRevision)
	s.Equal("insert", entries[0].ChangeHistoryAction)
	s.Equal("foo1", *entries[0].Foo)
	s.Equal("john@example.com", *entries[0].ChangeAuthor)

	s.Equal(2, entries[1].ChangeHistoryRevision)
	s.Equal("update", entries[1].ChangeHistoryAction)
	s.Equal("foo2", *entries[1].Foo)
	s.Equal("john@example.com", *entries[1].ChangeAuthor)

	s.Equal(3, entries[2].ChangeHistoryRevision)
	s.Equal("delete", entries[2].ChangeHistoryAction)
	s.Equal("foo2", *entries[2].Foo)
	s.Nil(entries[2].ChangeAuthor) // change-author is excluded for delete actions as it may be misleading
}

func TestChangeHistoryTestSuite(t *testing.T) {
	suite.Run(t, new(ChangeHistoryTestSuite))
}
