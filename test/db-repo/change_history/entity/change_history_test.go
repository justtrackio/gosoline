//go:build integration && fixtures

package change_history_test

import (
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/db-repo"
	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/test/suite"
	"github.com/justtrackio/gosoline/test/db-repo/change_history/entity/definitions"
)

type ChangeHistoryTestSuite struct {
	suite.Suite

	clock clock.Clock
}

func TestChangeHistoryTestSuite(t *testing.T) {
	suite.Run(t, new(ChangeHistoryTestSuite))
}

func (s *ChangeHistoryTestSuite) SetupSuite() []suite.Option {
	return []suite.Option{
		suite.WithLogLevel("info"),
		suite.WithConfigFile("config.test.yml"),
		suite.WithModule("default", definitions.ModuleFactory),
		suite.WithDbRepoChangeHistory(),
		suite.WithClockProviderAt("2024-01-01T00:00:00Z"),
		suite.WithContainerExpireAfter(2 * time.Minute),
		suite.WithFixtureSetFactory(definitions.FixtureSetsFactory),
	}
}

func (s *ChangeHistoryTestSuite) SetupTest() error {
	s.clock = clock.Provider

	return nil
}

func (s *ChangeHistoryTestSuite) Test_Create(app suite.AppUnderTest) {
	items := []definitions.Item{
		{
			Model: db_repo.Model{
				Id: mdl.Box(uint(1)),
			},
			Action: "create",
			Name:   "foo",
			ChangeHistoryEmbeddable: db_repo.ChangeHistoryEmbeddable{
				ChangeHistoryAuthorId: mdl.Box(uint(1)),
			},
		},
	}

	s.Env().StreamInput("default").PublishAndStop(items, map[string]string{})

	app.WaitDone()

	expected := []definitions.ItemsHistory{
		{
			ChangeHistoryModel: db_repo.ChangeHistoryModel{
				ChangeHistoryRevision: 1,
				ChangeHistoryActionAt: s.clock.Now(),
				ChangeHistoryAction:   "insert",
				ChangeHistoryAuthorId: 1,
			},
			Item: definitions.Item{
				Model: db_repo.Model{
					Id: mdl.Box(uint(1)),
					Timestamps: db_repo.Timestamps{
						CreatedAt: mdl.Box(s.clock.Now()),
						UpdatedAt: mdl.Box(s.clock.Now()),
					},
				},
				ChangeHistoryEmbeddable: db_repo.ChangeHistoryEmbeddable{},
				Action:                  "create",
				Name:                    "foo",
			},
		},
	}

	s.assertHistory(1, expected)
}

func (s *ChangeHistoryTestSuite) Test_Update(app suite.AppUnderTest) {
	items := []definitions.Item{
		{
			Model: db_repo.Model{
				Id: mdl.Box(uint(2)),
			},
			Action: "update",
			Name:   "bar",
			ChangeHistoryEmbeddable: db_repo.ChangeHistoryEmbeddable{
				ChangeHistoryAuthorId: mdl.Box(uint(1)),
			},
		},
	}

	s.Env().StreamInput("default").PublishAndStop(items, map[string]string{})

	app.WaitDone()

	expected := []definitions.ItemsHistory{
		{
			ChangeHistoryModel: db_repo.ChangeHistoryModel{
				ChangeHistoryRevision: 1,
				ChangeHistoryActionAt: s.clock.Now(),
				ChangeHistoryAction:   "update",
				ChangeHistoryAuthorId: 1,
			},
			Item: definitions.Item{
				Model: db_repo.Model{
					Id: mdl.Box(uint(2)),
					Timestamps: db_repo.Timestamps{
						CreatedAt: mdl.Box(s.clock.Now()),
						UpdatedAt: mdl.Box(s.clock.Now()),
					},
				},
				ChangeHistoryEmbeddable: db_repo.ChangeHistoryEmbeddable{},
				Action:                  "update",
				Name:                    "bar",
			},
		},
	}

	s.assertHistory(2, expected)
}

func (s *ChangeHistoryTestSuite) Test_Delete(app suite.AppUnderTest) {
	items := []definitions.Item{
		{
			Model: db_repo.Model{
				Id: mdl.Box(uint(3)),
			},
			Action: "delete",
			ChangeHistoryEmbeddable: db_repo.ChangeHistoryEmbeddable{
				ChangeHistoryAuthorId: mdl.Box(uint(1)),
			},
		},
	}

	s.Env().StreamInput("default").PublishAndStop(items, map[string]string{})

	app.WaitDone()

	expected := []definitions.ItemsHistory{
		{
			ChangeHistoryModel: db_repo.ChangeHistoryModel{
				ChangeHistoryRevision: 1,
				ChangeHistoryActionAt: s.clock.Now(),
				ChangeHistoryAction:   "delete",
				ChangeHistoryAuthorId: 1,
			},
			Item: definitions.Item{
				Model: db_repo.Model{
					Id: mdl.Box(uint(3)),
					Timestamps: db_repo.Timestamps{
						CreatedAt: mdl.Box(s.clock.Now()),
						UpdatedAt: mdl.Box(s.clock.Now()),
					},
				},
				ChangeHistoryEmbeddable: db_repo.ChangeHistoryEmbeddable{},
				Action:                  "delete",
				Name:                    "foo",
			},
		},
	}

	s.assertHistory(3, expected)
}

func (s *ChangeHistoryTestSuite) Test_MultipleManagers(app suite.AppUnderTest) {
	items := []definitions.Item{
		{
			Model: db_repo.Model{
				Id: mdl.Box(uint(4)),
			},
			Action: "create",
			Name:   "foo",
			ChangeHistoryEmbeddable: db_repo.ChangeHistoryEmbeddable{
				ChangeHistoryAuthorId: mdl.Box(uint(1)),
			},
		},
		{
			Model: db_repo.Model{
				Id: mdl.Box(uint(4)),
			},
			Action: "update",
			Name:   "bar",
			ChangeHistoryEmbeddable: db_repo.ChangeHistoryEmbeddable{
				ChangeHistoryAuthorId: mdl.Box(uint(2)),
			},
		},
		{
			Model: db_repo.Model{
				Id: mdl.Box(uint(4)),
			},
			Action: "delete",
			ChangeHistoryEmbeddable: db_repo.ChangeHistoryEmbeddable{
				ChangeHistoryAuthorId: mdl.Box(uint(3)),
			},
		},
	}

	s.Env().StreamInput("default").PublishAndStop(items, map[string]string{})

	app.WaitDone()

	expected := []definitions.ItemsHistory{
		{
			ChangeHistoryModel: db_repo.ChangeHistoryModel{
				ChangeHistoryRevision: 1,
				ChangeHistoryActionAt: s.clock.Now(),
				ChangeHistoryAction:   "insert",
				ChangeHistoryAuthorId: 1,
			},
			Item: definitions.Item{
				Model: db_repo.Model{
					Id: mdl.Box(uint(4)),
					Timestamps: db_repo.Timestamps{
						CreatedAt: mdl.Box(s.clock.Now()),
						UpdatedAt: mdl.Box(s.clock.Now()),
					},
				},
				ChangeHistoryEmbeddable: db_repo.ChangeHistoryEmbeddable{},
				Action:                  "create",
				Name:                    "foo",
			},
		},
		{
			ChangeHistoryModel: db_repo.ChangeHistoryModel{
				ChangeHistoryRevision: 2,
				ChangeHistoryActionAt: s.clock.Now(),
				ChangeHistoryAction:   "update",
				ChangeHistoryAuthorId: 2,
			},
			Item: definitions.Item{
				Model: db_repo.Model{
					Id: mdl.Box(uint(4)),
					Timestamps: db_repo.Timestamps{
						CreatedAt: mdl.Box(s.clock.Now()),
						UpdatedAt: mdl.Box(s.clock.Now()),
					},
				},
				ChangeHistoryEmbeddable: db_repo.ChangeHistoryEmbeddable{},
				Action:                  "update",
				Name:                    "bar",
			},
		},
		{
			ChangeHistoryModel: db_repo.ChangeHistoryModel{
				ChangeHistoryRevision: 3,
				ChangeHistoryActionAt: s.clock.Now(),
				ChangeHistoryAction:   "delete",
				ChangeHistoryAuthorId: 3,
			},
			Item: definitions.Item{
				Model: db_repo.Model{
					Id: mdl.Box(uint(4)),
					Timestamps: db_repo.Timestamps{
						CreatedAt: mdl.Box(s.clock.Now()),
						UpdatedAt: mdl.Box(s.clock.Now()),
					},
				},
				ChangeHistoryEmbeddable: db_repo.ChangeHistoryEmbeddable{},
				Action:                  "update",
				Name:                    "bar",
			},
		},
	}

	s.assertHistory(4, expected)
}

func (s *ChangeHistoryTestSuite) assertHistory(entityId uint, expected []definitions.ItemsHistory) {
	repo, err := definitions.NewHistoryRepository(s.Env().Context(), s.Env().Config(), s.Env().Logger())
	if !s.NoError(err) {
		return
	}

	histories := make([]*definitions.ItemsHistory, 0)

	qb := db_repo.NewQueryBuilder()
	qb.Where("id = ?", entityId)

	err = repo.Query(s.Env().Context(), qb, &histories)
	if !s.NoError(err) {
		return
	}

	items := funk.Map(histories, func(h *definitions.ItemsHistory) definitions.ItemsHistory {
		// modify timestamp as we can't match the current assigned by the database
		h.ChangeHistoryActionAt = s.clock.Now()

		return *h
	})

	s.Equal(expected, items)
}
