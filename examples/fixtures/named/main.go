//go:build fixtures

package main

import (
	"github.com/justtrackio/gosoline/pkg/application"
	"github.com/justtrackio/gosoline/pkg/db-repo"
	"github.com/justtrackio/gosoline/pkg/fixtures"
	"github.com/justtrackio/gosoline/pkg/mdl"
)

type DynamoDbExampleModel struct {
	Name  string `ddb:"key=hash"`
	Value string `ddb:"global=hash"`
}

func main() {
	// store named fixtures
	namedFixtures := &namedFixtureBuilder{}

	app := application.Default(
		application.WithFixtureBuilderFactory(fixtures.SimpleFixtureBuilderFactory(namedFixtures.Fixtures())),
	)

	app.Run()

	// then you can access them later
	fx := namedFixtures.GetNamed("test")
	_ = fx.Value
}

type namedFixtureBuilder struct {
	fixtures fixtures.NamedFixtureSet
}

func (b *namedFixtureBuilder) Fixtures() []*fixtures.FixtureSet {
	b.fixtures = fixtures.NamedFixtureSet{
		{
			Name:  "test",
			Value: &DynamoDbExampleModel{Name: "Some Name", Value: "Some Value"},
		},
	}

	return []*fixtures.FixtureSet{
		{
			Enabled: true,
			Purge:   false,
			Writer: fixtures.MysqlOrmFixtureWriterFactory(&db_repo.Metadata{
				ModelId: mdl.ModelId{
					Name: "orm_named_fixture_example",
				},
			}),
			Fixtures: b.fixtures.All(),
		},
	}
}

// GetNamed Add properly typed getter
func (b *namedFixtureBuilder) GetNamed(name string) *DynamoDbExampleModel {
	return b.fixtures.GetValueByName(name).(*DynamoDbExampleModel)
}
