package integration

import (
	"github.com/justtrackio/gosoline/pkg/fixtures/writers"
	"github.com/justtrackio/gosoline/pkg/fixtures/writers/kvstore"
)

type NamedFixture struct {
	Name  string
	Value interface{}
}

var fixtureSets = []*writers.FixtureSet{
	{
		Enabled: true,
		Writer:  kvstore.ConfigurableKvStoreFixtureWriterFactory("currency"),
		Fixtures: []interface{}{
			&kvstore.KvStoreFixture{
				Key:   "GBP",
				Value: 1.25,
			},
			&kvstore.KvStoreFixture{
				Key:   "2021-01-03-GBP",
				Value: 0.8,
			},
		},
	},
}
