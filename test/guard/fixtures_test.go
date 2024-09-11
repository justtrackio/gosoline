//go:build integration && fixtures

package guard_test

import (
	"github.com/justtrackio/gosoline/pkg/fixtures"
)

var metadata = &fixtures.MysqlPlainMetaData{
	TableName: "guard_policies",
	Columns:   []string{"id", "policy"},
}

var namedFixtures = fixtures.NamedFixtures[fixtures.MysqlPlainFixtureValues]{
	{
		Name: "fixture_1",
		Value: fixtures.MysqlPlainFixtureValues{
			"a97b104f-4d93-4c15-a97a-4e3173e75cde",
			`{
				"id": "a97b104f-4d93-4c15-a97a-4e3173e75cde",
				"effect": "allow",
				"actions": ["<.+>"],
				"subjects": ["r:1"],
				"resources": ["gsl:<.+>"],
				"conditions": {},
				"description": "global - read and write access"
			}`,
		},
	},
	{
		Name: "fixture_2",
		Value: fixtures.MysqlPlainFixtureValues{
			"18a1de65-62eb-4af6-aab4-593d05ed30be",
			`{
				"id": "18a1de65-62eb-4af6-aab4-593d05ed30be",
				"effect": "allow",
				"actions": ["<.+>"],
				"subjects": ["r:2"],
				"resources": ["gsl:e:1:<.+>"],
				"conditions": {},
				"description": "entity - read and write access"
			}`,
		},
	},
	{
		Name: "fixture_3",
		Value: fixtures.MysqlPlainFixtureValues{
			"4ab80e96-22ea-469e-96d1-12b232bd4660",
			`{
				"id": "4ab80e96-22ea-469e-96d1-12b232bd4660",
				"effect": "allow",
				"actions": ["read"],
				"subjects": [],
				"resources": [],
				"conditions": {},
				"description": "global - read access"
			}`,
		},
	},
}
