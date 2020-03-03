package main

import (
	"github.com/applike/gosoline/pkg/application"
	"github.com/applike/gosoline/pkg/db-repo"
	"github.com/applike/gosoline/pkg/ddb"
	"github.com/applike/gosoline/pkg/fixtures"
	"github.com/applike/gosoline/pkg/mdl"
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

func createFixtures() []*fixtures.FixtureSet {
	return []*fixtures.FixtureSet{
		{
			Enabled: true,
			Writer: fixtures.MysqlOrmFixtureWriterFactory(
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
			Writer: fixtures.MysqlPlainFixtureWriterFactory(&fixtures.MysqlPlainMetaData{
				TableName: "plain_fixture_example",
				Columns:   []string{"id", "name"},
			}),
			Fixtures: []interface{}{
				&fixtures.MysqlPlainFixtureValues{1, "testName1"},
				&fixtures.MysqlPlainFixtureValues{2, "testName2"},
			},
		},
		{
			Enabled: true,
			Writer: fixtures.DynamoDbKvStoreFixtureWriterFactory(&mdl.ModelId{
				Project:     "gosoline",
				Environment: "dev",
				Family:      "example",
				Application: "fixture-loader",
				Name:        "exampleModel",
			}),
			Fixtures: []interface{}{
				&fixtures.KvStoreFixture{
					Key:   "SomeKey",
					Value: &DynamoDbExampleModel{Name: "Some Name", Value: "Some Value"},
				},
			},
		},
		{
			Enabled: true,
			Writer: fixtures.DynamoDbFixtureWriterFactory(&ddb.Settings{
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
	}
}
