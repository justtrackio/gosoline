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

type DbRepoChangeHistoryTestSuite struct {
	suite.Suite
	logger mon.Logger
	config cfg.Config
	mocks  *test.Mocks
	repo   db_repo.Repository
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
			"db_port":     s.mocks.ProvideMysqlPort("mysql"),
			"db_hostname": s.mocks.ProvideMysqlHost("mysql"),
		}),
	)

	s.config = config
	s.logger = mon.NewLogger()
	s.repo = db_repo.New(s.config, s.logger, db_repo.Settings{})
}

func (s *DbRepoChangeHistoryTestSuite) TearDownSuite() {
	s.mocks.Shutdown()
}

func (s *DbRepoChangeHistoryTestSuite) TestChangeHistoryMigration_Migrate_CreateTable() {

	type TestModel1 struct {
		db_repo.Model
		Name *string
	}

	type TestModel1HistoryEntry struct {
		db_repo.ChangeHistoryModel
		TestModel1
	}

	s.NotPanics(func() {
		db_repo.MigrateChangeHistory(s.config, s.logger, &TestModel1{})
	})

	model := &TestModel1{
		Name: mdl.String("name1"),
	}

	err := s.repo.Create(context.Background(), model)
	s.NoError(err)

	model.Name = mdl.String("name2")
	err = s.repo.Update(context.Background(), model)
	s.NoError(err)

	err = s.repo.Delete(context.Background(), model)
	s.NoError(err)

	entries := make([]*TestModel1HistoryEntry, 0)
	err = s.repo.Query(context.Background(), &db_repo.QueryBuilder{}, &entries)
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

	s.NotPanics(func() {
		db_repo.MigrateChangeHistory(s.config, s.logger, &TestModel2{})
	})

	model := &TestModel2{
		Name:         mdl.String("name1"),
		Foo:          mdl.String("foo1"),
		ChangeAuthor: mdl.String("john@example.com"),
	}

	err := s.repo.Create(context.Background(), model)
	s.NoError(err)

	model.Foo = mdl.String("foo2")
	err = s.repo.Update(context.Background(), model)
	s.NoError(err)

	err = s.repo.Delete(context.Background(), model)
	s.NoError(err)

	entries := make([]*TestModel2HistoryEntry, 0)
	err = s.repo.Query(context.Background(), &db_repo.QueryBuilder{}, &entries)
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
