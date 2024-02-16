// snippet-start: imports
package apitest

import "github.com/justtrackio/gosoline/pkg/fixtures"

// snippet-end: imports

// snippet-start: fixture set
var fixtureSets = []*fixtures.FixtureSet{
	{
		Enabled: true,
		Writer:  fixtures.ConfigurableKvStoreFixtureWriterFactory[float64]("currency"),
		Fixtures: []interface{}{
			&fixtures.KvStoreFixture{
				Key:   "GBP",
				Value: 1.25,
			},
			&fixtures.KvStoreFixture{
				Key:   "2021-01-03-GBP",
				Value: 0.8,
			},
		},
	},
}

// snippet-end: fixture set
