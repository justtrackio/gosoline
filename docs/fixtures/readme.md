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
	Writer   FixtureWriterFactory
	Fixtures []interface{}
}
```
You can easily define multiple fixtures to different destinations in one file and enable or disable them for each FixtureWriter.

* Currently there are 4 different FixtureWriterFactories implemented to load fixtures. 
    * `DynamoDbFixtureWriterFactory`
    * `DynamoDbKvStoreFixtureWriterFactory` 
    * `MysqlOrmFixtureWriterFactory` 
    * `MysqlPlainFixtureWriterFactory` 

## Quick Usage
* During the creation of your Application make sure to pass the `WithFixtures` option and provide fixtures as an argument of type `[]*fixtures.FixtureSet`
```
//+build fixtures

package main

import ...

type OrmFixtureExample struct {
	db_repo.Model
	Name *string
}

func main() {
	app := application.Default(application.WithFixtures(
		[]*fixtures.FixtureSet{
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
		},))

	app.Run()
}
```

## Further Information
* Existing fixtures will be updated instead of created.
* If you want to use the `MysqlOrmFixtureWriterFactory` make sure that your fixture struct embeds `db_repo.Model`   
* For example usage of all `FixtureWriterFactory` see the `/examples/gosoline-fixture-loading` directory