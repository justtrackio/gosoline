//go:build fixtures

package main

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/justtrackio/gosoline/pkg/application"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/db"
	"github.com/justtrackio/gosoline/pkg/fixtures"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
)

type User struct {
	Id       int    `db:"id"`
	Name     string `db:"name"`
	IsActive bool   `db:"is_active"`
}

func fixtureSetsFactory(ctx context.Context, config cfg.Config, logger log.Logger, group string) ([]fixtures.FixtureSet, error) {
	writer, err := fixtures.NewMysqlSqlxFixtureWriter(ctx, config, logger, &fixtures.MysqlSqlxMetaData{TableName: "users"})
	if err != nil {
		return nil, fmt.Errorf("failed to provide writers: %w", err)
	}

	fs := fixtures.NewSimpleFixtureSet(fixtures.NamedFixtures[User]{
		&fixtures.NamedFixture[User]{
			Name: "bob",
			Value: User{
				Id:       1,
				Name:     "Bob",
				IsActive: true,
			},
		},
		&fixtures.NamedFixture[User]{
			Name: "alice",
			Value: User{
				Id:       2,
				Name:     "Alice",
				IsActive: true,
			},
		},
	}, writer, fixtures.WithPurge(true))

	return []fixtures.FixtureSet{
		fs,
	}, nil
}

func main() {
	application.RunFunc(
		func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.ModuleRunFunc, error) {
			var err error
			var client db.Client

			if client, err = db.NewClient(ctx, config, logger, "default"); err != nil {
				return nil, err
			}

			return func(ctx context.Context) error {
				var rows *sqlx.Rows
				user := &User{}

				if rows, err = client.Queryx(ctx, "SELECT * FROM users"); err != nil {
					return err
				}

				for rows.Next() {
					if err := rows.StructScan(&user); err != nil {
						return err
					}

					fmt.Printf("%#v\n", user)
				}

				return nil
			}, nil
		},
		application.WithConfigFile("config.dist.yml", "yaml"),
		application.WithFixtureSetFactory("default", fixtureSetsFactory),
	)
}
