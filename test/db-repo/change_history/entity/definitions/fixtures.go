package definitions

import (
	"github.com/justtrackio/gosoline/pkg/db-repo"
	"github.com/justtrackio/gosoline/pkg/fixtures"
	"github.com/justtrackio/gosoline/pkg/mdl"
)

var FixtureSets = []*fixtures.FixtureSet{
	{
		Enabled: true,
		Writer:  fixtures.MysqlOrmFixtureWriterFactory(&tableMetadata),
		Fixtures: []interface{}{
			&Item{
				Model: db_repo.Model{
					Id: mdl.Box(uint(2)),
				},
				ChangeHistoryEmbeddable: db_repo.ChangeHistoryEmbeddable{},
				Action:                  "update",
				Name:                    "foo",
			},
			&Item{
				Model: db_repo.Model{
					Id: mdl.Box(uint(3)),
				},
				ChangeHistoryEmbeddable: db_repo.ChangeHistoryEmbeddable{},
				Action:                  "delete",
				Name:                    "foo",
			},
		},
	},
}
