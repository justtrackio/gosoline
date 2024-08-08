//go:build fixtures

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/justtrackio/gosoline/pkg/application"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/db-repo"
	"github.com/justtrackio/gosoline/pkg/ddb"
	"github.com/justtrackio/gosoline/pkg/fixtures"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
)

type DynamoDbExampleModel struct {
	Name  string `ddb:"key=hash"`
	Value string `ddb:"global=hash"`
}

type OrmFixtureExample struct {
	db_repo.Model
	Name *string
}

func main() {
	app := application.Default(
		application.WithFixtureSetFactory(provideFixtureSets),
	)

	app.Run()
}

var autoNumbered = fixtures.NewAutoNumberedFrom(2)

func provideFixtureSets(ctx context.Context, config cfg.Config, logger log.Logger) ([]fixtures.FixtureSet, error) {
	mysqlMetadata := &db_repo.Metadata{
		ModelId: mdl.ModelId{
			Name: "orm_fixture_example",
		},
	}
	mysqlOrmWriter, err := fixtures.NewMysqlOrmFixtureWriter(ctx, config, logger, mysqlMetadata)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize mysql orm writer: %w", err)
	}

	return []fixtures.FixtureSet{
		&fixtures.SimpleFixtureSet{
			Enabled: true,
			Writer:  mysqlOrmWriter,
			Fixtures: []interface{}{
				&OrmFixtureExample{
					Model: db_repo.Model{
						Id: autoNumbered.GetNext(),
					},
					Name: mdl.Box("example"),
				},
			},
		},
		&fixtures.SimpleFixtureSet{
			Enabled: true,
			Writer: fixtures.NewMysqlPlainFixtureWriter(&fixtures.MysqlPlainMetaData{
				TableName: "plain_fixture_example",
				Columns:   []string{"id", "name"},
			}),
			Fixtures: []interface{}{
				fixtures.MysqlPlainFixtureValues{1, "testName1"},
				fixtures.MysqlPlainFixtureValues{2, "testName2"},
			},
		},
		&fixtures.SimpleFixtureSet{
			Enabled: true,
			Writer: fixtures.NewDynamoDbKvStoreFixtureWriter[DynamoDbExampleModel](&mdl.ModelId{
				Project:     "gosoline",
				Environment: "dev",
				Family:      "example",
				Application: "fixture-loader",
				Name:        "exampleModel",
			}),
			Fixtures: []interface{}{
				&fixtures.KvStoreFixture{
					Key:   "SomeKey",
					Value: DynamoDbExampleModel{Name: "Some Name", Value: "Some Value"},
				},
			},
		},
		&fixtures.SimpleFixtureSet{
			Enabled: true,
			Purge:   true,
			Writer:  fixtures.NewRedisFixtureWriter(aws.String("default"), aws.String(fixtures.RedisOpSet)),
			Fixtures: []interface{}{
				&fixtures.RedisFixture{
					Key:    "example-key",
					Value:  "bar",
					Expiry: 1 * time.Hour,
				},
			},
		},
		&fixtures.SimpleFixtureSet{
			Enabled: true,
			Writer: fixtures.NewDynamoDbFixtureWriter(&ddb.Settings{
				ModelId: mdl.ModelId{
					Project:     "gosoline",
					Environment: "dev",
					Family:      "example",
					Application: "fixture-loader",
					Name:        "exampleModel",
				},
				Main: ddb.MainSettings{
					Model: DynamoDbExampleModel{},
				},
				Global: []ddb.GlobalSettings{
					{
						Name:               "IDX_Name",
						Model:              DynamoDbExampleModel{},
						ReadCapacityUnits:  1,
						WriteCapacityUnits: 1,
					},
				},
			}),
			Fixtures: []interface{}{
				&DynamoDbExampleModel{Name: "Some Name", Value: "Some Value"},
			},
		},
		&fixtures.SimpleFixtureSet{
			Enabled: true,
			Purge:   false,
			Writer: fixtures.NewBlobFixtureWriter(&fixtures.BlobFixturesSettings{
				ConfigName: "test",
				BasePath:   "../../test/test_data/s3_fixtures_test_data",
			}),
			Fixtures: nil,
		},
	}, nil
}
