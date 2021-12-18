package main

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/justtrackio/gosoline/pkg/application"
	"github.com/justtrackio/gosoline/pkg/db-repo"
	"github.com/justtrackio/gosoline/pkg/ddb"
	"github.com/justtrackio/gosoline/pkg/fixtures/writers"
	ddbFixtures "github.com/justtrackio/gosoline/pkg/fixtures/writers/ddb"
	"github.com/justtrackio/gosoline/pkg/fixtures/writers/mysql"
	"github.com/justtrackio/gosoline/pkg/fixtures/writers/redis"
	"github.com/justtrackio/gosoline/pkg/fixtures/writers/s3"
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
	app := application.Default(application.WithFixtures(createFixtures()))
	app.Run()
}

func createFixtures() []*writers.FixtureSet {
	return []*writers.FixtureSet{
		{
			Enabled: true,
			Writer: mysql.MysqlOrmFixtureWriterFactory(
				&db_repo.Metadata{
					ModelId: mdl.ModelId{
						Name: "orm_fixture_example",
					},
				},
			),
			Fixtures: []interface{}{
				&OrmFixtureExample{
					Model: db_repo.Model{
						Id: mdl.Uint(1),
					},
					Name: mdl.String("example"),
				},
			},
		},
		{
			Enabled: true,
			Writer: mysql.MysqlPlainFixtureWriterFactory(&mysql.MysqlPlainMetaData{
				TableName: "plain_fixture_example",
				Columns:   []string{"id", "name"},
			}),
			Fixtures: []interface{}{
				mysql.MysqlPlainFixtureValues{1, "testName1"},
				mysql.MysqlPlainFixtureValues{2, "testName2"},
			},
		},
		{
			Enabled: true,
			Writer: ddbFixtures.DynamoDbKvStoreFixtureWriterFactory(&mdl.ModelId{
				Project:     "gosoline",
				Environment: "dev",
				Family:      "example",
				Application: "fixture-loader",
				Name:        "exampleModel",
			}),
			Fixtures: []interface{}{
				&ddbFixtures.KvStoreFixture{
					Key:   "SomeKey",
					Value: &DynamoDbExampleModel{Name: "Some Name", Value: "Some Value"},
				},
			},
		},
		{
			Enabled: true,
			Purge:   true,
			Writer:  redis.RedisFixtureWriterFactory(aws.String("default"), aws.String(redis.RedisOpSet)),
			Fixtures: []interface{}{
				&redis.RedisFixture{
					Key:    "example-key",
					Value:  "bar",
					Expiry: 1 * time.Hour,
				},
			},
		},
		{
			Enabled: true,
			Writer: ddbFixtures.DynamoDbFixtureWriterFactory(&ddb.Settings{
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
		{
			Enabled: true,
			Purge:   false,
			Writer: s3.BlobFixtureWriterFactory(&s3.BlobFixturesSettings{
				ConfigName: "test",
				BasePath:   "../../test/test_data/s3_fixtures_test_data",
			}),
			Fixtures: nil,
		},
	}
}
