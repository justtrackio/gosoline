package definitions

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/db-repo"
	"github.com/justtrackio/gosoline/pkg/fixtures"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
)

var repoFixtures = fixtures.NamedFixtures[*Item]{
	{
		Name: "foo_update_item",
		Value: &Item{
			Model: db_repo.Model{
				Id: mdl.Box(uint(2)),
			},
			ChangeHistoryEmbeddable: db_repo.ChangeHistoryEmbeddable{},
			Action:                  "update",
			Name:                    "foo",
		},
	},
	{
		Name: "foo_delete_item",
		Value: &Item{
			Model: db_repo.Model{
				Id: mdl.Box(uint(3)),
			},
			ChangeHistoryEmbeddable: db_repo.ChangeHistoryEmbeddable{},
			Action:                  "delete",
			Name:                    "foo",
		},
	},
}

func FixtureSetsFactory(ctx context.Context, config cfg.Config, logger log.Logger, group string) ([]fixtures.FixtureSet, error) {
	writer, err := fixtures.NewMysqlOrmFixtureWriter(ctx, config, logger, &tableMetadata)
	if err != nil {
		return nil, fmt.Errorf("failed to create kvstore fixture writer: %w", err)
	}

	fss := []fixtures.FixtureSet{
		fixtures.NewSimpleFixtureSet(repoFixtures, writer),
	}

	return fss, nil
}
