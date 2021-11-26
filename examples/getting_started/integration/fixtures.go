package integration

import (
	gosoFixtures "github.com/justtrackio/gosoline/pkg/fixtures"
)

type NamedFixture struct {
	Name  string
	Value interface{}
}

var fixtureSets = []*gosoFixtures.FixtureSet{
	{
		Enabled: true,
		Writer:  gosoFixtures.ConfigurableKvStoreFixtureWriterFactory("currency"),
		Fixtures: []interface{}{
			&gosoFixtures.KvStoreFixture{
				Key:   "GBP",
				Value: 1.25,
			},
			&gosoFixtures.KvStoreFixture{
				Key:   "2021-01-03-GBP",
				Value: 0.8,
			},
		},
	},
}
