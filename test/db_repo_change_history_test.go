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

type DbRepoChangeHistoryTestSuite struct {
	suite.Suite
	logger           log.Logger
	config           cfg.Config
	mocks            *test.Mocks
	modelRepo        db_repo.Repository
	modelHistoryRepo db_repo.Repository
}

func TestDbChangelogTestSuite(t *testing.T) {
	suite.Run(t, new(DbRepoChangeHistoryTestSuite))
}

func (s *DbRepoChangeHistoryTestSuite) SetupSuite() {
	setup(s.T())
	mocks, err := test.Boot("test_configs/config.mysql.test.yml")

	if !s.NoError(err) {
		return
	}

	s.mocks = mocks

	config := cfg.New()
	_ = config.Option(
		cfg.WithConfigFile("test_configs/config.mysql.test.yml", "yml"),
		cfg.WithConfigFile("test_configs/config.db_repo_change_history.test.yml", "yml"),
		cfg.WithConfigMap(map[string]interface{}{
			"db.default.uri.port": s.mocks.ProvideMysqlPort("mysql"),
			"db.default.uri.host": s.mocks.ProvideMysqlHost("mysql"),
		}),
	)

	s.config = config
	s.logger = log.NewCliLogger()
}

func (s *DbRepoChangeHistoryTestSuite) TearDownSuite() {
	s.mocks.Shutdown()
}

func (s *DbRepoChangeHistoryTestSuite) TestChangeHistoryMigration_Migrate_CreateTable() {
	var err error

	s.modelRepo, err = db_repo.New(s.config, s.logger, db_repo.Settings{
		Metadata: TestModel1Metadata,
	})
	s.NoError(err)

	s.modelHistoryRepo, err = db_repo.New(s.config, s.logger, db_repo.Settings{
		Metadata: TestHistoryModel1Metadata,
	})
	s.NoError(err)

	err = db_repo.MigrateChangeHistory(s.config, s.logger, &TestModel1{})
	s.NoError(err)

	model := &TestModel1{
		Name: mdl.String("name1"),
	}

	err = s.modelRepo.Create(context.Background(), model)
	s.NoError(err)

	model.Name = mdl.String("name2")
	err = s.modelRepo.Update(context.Background(), model)
	s.NoError(err)

	err = s.modelRepo.Delete(context.Background(), model)
	s.NoError(err)

	entries := make([]*TestModel1HistoryEntry, 0)
	err = s.modelHistoryRepo.Query(context.Background(), &db_repo.QueryBuilder{}, &entries)
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

func (s *DbRepoChangeHistoryTestSuite) TestChangeHistoryMigration_Migrate_UpdateTable() {
	var err error

	s.modelRepo, err = db_repo.New(s.config, s.logger, db_repo.Settings{
		Metadata: TestModel2Metadata,
	})
	s.NoError(err)

	s.modelHistoryRepo, err = db_repo.New(s.config, s.logger, db_repo.Settings{
		Metadata: TestHistoryModel2Metadata,
	})
	s.NoError(err)

	err = db_repo.MigrateChangeHistory(s.config, s.logger, &TestModel2{})
	s.NoError(err)

	model := &TestModel2{
		Name:         mdl.String("name1"),
		Foo:          mdl.String("foo1"),
		ChangeAuthor: mdl.String("john@example.com"),
	}

	err = s.modelRepo.Create(context.Background(), model)
	s.NoError(err)

	model.Foo = mdl.String("foo2")
	err = s.modelRepo.Update(context.Background(), model)
	s.NoError(err)

	err = s.modelRepo.Delete(context.Background(), model)
	s.NoError(err)

	entries := make([]*TestModel2HistoryEntry, 0)
	err = s.modelHistoryRepo.Query(context.Background(), &db_repo.QueryBuilder{}, &entries)
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
