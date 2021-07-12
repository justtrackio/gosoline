// +build integration,fixtures

package guard_test

import "github.com/applike/gosoline/pkg/fixtures"

func buildPolicies() []interface{} {
	return []interface{}{
		fixtures.MysqlPlainFixtureValues{"a97b104f-4d93-4c15-a97a-4e3173e75cde", "global - read and write access", "allow"},
		fixtures.MysqlPlainFixtureValues{"18a1de65-62eb-4af6-aab4-593d05ed30be", "entity - read and write access", "allow"},
		fixtures.MysqlPlainFixtureValues{"4ab80e96-22ea-469e-96d1-12b232bd4660", "global - read access", "allow"},
	}
}

func buildSubjects() []interface{} {
	return []interface{}{
		fixtures.MysqlPlainFixtureValues{"a97b104f-4d93-4c15-a97a-4e3173e75cde", "r:1"},
		fixtures.MysqlPlainFixtureValues{"18a1de65-62eb-4af6-aab4-593d05ed30be", "r:2"},
	}
}

func buildResources() []interface{} {
	return []interface{}{
		fixtures.MysqlPlainFixtureValues{"a97b104f-4d93-4c15-a97a-4e3173e75cde", "gsl:<.+>"},
		fixtures.MysqlPlainFixtureValues{"18a1de65-62eb-4af6-aab4-593d05ed30be", "gsl:e:1:<.+>"},
	}
}

func buildActions() []interface{} {
	return []interface{}{
		fixtures.MysqlPlainFixtureValues{"a97b104f-4d93-4c15-a97a-4e3173e75cde", "<.+>"},
		fixtures.MysqlPlainFixtureValues{"18a1de65-62eb-4af6-aab4-593d05ed30be", "<.+>"},
		fixtures.MysqlPlainFixtureValues{"ab80e96-22ea-469e-96d1-12b232bd4660", "read"},
	}
}

func buildFixtures() []*fixtures.FixtureSet {
	return []*fixtures.FixtureSet{
		{
			Enabled: true,
			Purge:   false,
			Writer: fixtures.MysqlPlainFixtureWriterFactory(&fixtures.MysqlPlainMetaData{
				TableName: "guard_policies",
				Columns:   []string{"id", "description", "effect"},
			}),
			Fixtures: buildPolicies(),
		},
		{
			Enabled: true,
			Purge:   false,
			Writer: fixtures.MysqlPlainFixtureWriterFactory(&fixtures.MysqlPlainMetaData{
				TableName: "guard_subjects",
				Columns:   []string{"id", "name"},
			}),
			Fixtures: buildSubjects(),
		},
		{
			Enabled: true,
			Purge:   false,
			Writer: fixtures.MysqlPlainFixtureWriterFactory(&fixtures.MysqlPlainMetaData{
				TableName: "guard_resources",
				Columns:   []string{"id", "name"},
			}),
			Fixtures: buildResources(),
		},
		{
			Enabled: true,
			Purge:   false,
			Writer: fixtures.MysqlPlainFixtureWriterFactory(&fixtures.MysqlPlainMetaData{
				TableName: "guard_actions",
				Columns:   []string{"id", "name"},
			}),
			Fixtures: buildActions(),
		},
	}
}
