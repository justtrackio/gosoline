# Gosoline Fixture Loader
While using *gosoline* you are able to define fixtures directly in code. 
 
## Usage
* Enable fixture loading in your `config.dist.yml` File
```
fixtures:
    enabled: true
```
* Make sure to use a custom build tag named `fixtures`

* Define your fixtures in code while using one of the built in `FixtureWriterFactory` and defining a slice of `[]fixtures.FixtureSet{}`.
For more details check the `/examples/gosoline-fixture-loading` Directory or read the short example below. 

* The `fixtures.FixtureSet{}` has the following definition:
```
type FixtureSet struct {
	Enabled  bool
	Purge    bool
	Writer   FixtureWriterFactory
	Fixtures []interface{}
}
```
You can easily define multiple fixtures to different destinations in one file and enable or disable them for each FixtureWriter.

* Currently there are 7 different FixtureWriterFactories implemented to load fixtures. 
    * `DynamoDbFixtureWriterFactory`
    * `DynamoDbKvStoreFixtureWriterFactory` 
    * `MysqlOrmFixtureWriterFactory` 
    * `MysqlPlainFixtureWriterFactory`
    * `RedisFixtureWriterFactory`
    * `RedisKvStoreFixtureWriterFactory` 
    * `BlobFixtureWriterFactory` 

## Quick Usage
* During the creation of your Application make sure to pass the `WithFixtures` option and provide fixtures as an argument of type `[]*fixtures.FixtureSet`

[embedmd]:# (../../examples/fixtures/main.go /func main/ /}/)
```go
func main() {
	app := application.Default(application.WithFixtures(createFixtures()))
	app.Run()
}
```

or for example when using the `BlobFixtureWriterFactory`:

```yaml
blobstore:
  blobconfig:
    bucket: s3-fixtures-bucket
```

[embedmd]:# (../../examples/fixtures/main.go /func main/ $)
```go
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
						Id: mdl.Box(uint(1)),
					},
					Name: mdl.Box("example"),
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
				fixtures.MysqlPlainFixtureValues{1, "testName1"},
				fixtures.MysqlPlainFixtureValues{2, "testName2"},
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
			Purge:   true,
			Writer:  fixtures.RedisFixtureWriterFactory(aws.String("default"), aws.String(fixtures.RedisOpSet)),
			Fixtures: []interface{}{
				&fixtures.RedisFixture{
					Key:    "example-key",
					Value:  "bar",
					Expiry: 1 * time.Hour,
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
		{
			Enabled: true,
			Purge:   false,
			Writer: fixtures.BlobFixtureWriterFactory(&fixtures.BlobFixturesSettings{
				ConfigName: "test",
				BasePath:   "../../test/test_data/s3_fixtures_test_data",
			}),
			Fixtures: nil,
		},
	}
}
```

## Further Information
* Existing fixtures will be updated instead of created.
* When purge is enabled only the destination of the fixtures will be purged, not everything. That means for example while loading MySQL Fixtures with purge only the tables will be purged not the whole database.  
* If you want to use the `MysqlOrmFixtureWriterFactory` make sure that your fixture struct embeds `db_repo.Model`   
* For example usage of all `FixtureWriterFactory` see the `/examples/gosoline-fixture-loading` directory
