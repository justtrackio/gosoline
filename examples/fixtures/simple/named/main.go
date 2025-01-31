//go:build fixtures

package main

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/application"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/db-repo"
	"github.com/justtrackio/gosoline/pkg/fixtures"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
)

type DynamoDbExampleModel struct {
	Name  string `ddb:"key=hash"`
	Value string `ddb:"global=hash"`
}

var namedFixtures = fixtures.NamedFixtures[*DynamoDbExampleModel]{
	{
		Name:  "test",
		Value: &DynamoDbExampleModel{Name: "Some Name", Value: "Some Value"},
	},
}

func fixtureSetsFactory(ctx context.Context, config cfg.Config, logger log.Logger, group string) ([]fixtures.FixtureSet, error) {
	mysqlWriter, err := db_repo.NewMysqlOrmFixtureWriter(ctx, config, logger, &db_repo.Metadata{
		ModelId: mdl.ModelId{
			Name: "orm_named_fixture_example",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create orm fixture writer: %w", err)
	}

	return []fixtures.FixtureSet{
		fixtures.NewSimpleFixtureSet(namedFixtures, mysqlWriter),
	}, nil
}

func main() {
	app := application.Default(
		application.WithFixtureSetFactory("default", fixtureSetsFactory),
	)

	app.Run()

	// then you can access them later
	fx := namedFixtures.GetValueByName("test")
	_ = fx.Value
}
