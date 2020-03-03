//+build integration

package test_test

import (
	"database/sql"
	db_repo "github.com/applike/gosoline/pkg/db-repo"
	"github.com/applike/gosoline/pkg/fixtures"
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/test"
	gosoAssert "github.com/applike/gosoline/pkg/test/assert"
	"github.com/stretchr/testify/assert"
	"testing"
)

type MysqlTestModel struct {
	db_repo.Model
	Name *string
}

var TestModelMetadata = db_repo.Metadata{
	ModelId: mdl.ModelId{
		Name: "test_model",
	},
	TableName:  "test_models",
	PrimaryKey: "test_model.id",
	Mappings: db_repo.FieldMappings{
		"test_model.id":   db_repo.NewFieldMapping("test_model.id"),
		"test_model.name": db_repo.NewFieldMapping("test_model.name"),
	},
}

func mysqlTestFixtures() []*fixtures.FixtureSet {
	return []*fixtures.FixtureSet{
		{
			Enabled: true,
			Writer:  fixtures.MysqlFixtureWriterFactory(&TestModelMetadata),
			Fixtures: []interface{}{
				&MysqlTestModel{
					Name: mdl.String("testName"),
				},
			},
		},
	}
}

func Test_enabled_fixtures_mysql(t *testing.T) {
	setup(t)

	configFile := "test_configs/config.mysql.test.yml"

	mocks := test.Boot(configFile)
	defer mocks.Shutdown()

	logger := mon.NewLogger()
	config := configFromFiles("test_configs/config.mysql.test.yml", "test_configs/config.fixtures_mysql.test.yml")
	loader := fixtures.NewFixtureLoader(config, logger)

	err := loader.Load(mysqlTestFixtures())
	assert.NoError(t, err)

	db := mocks.ProvideClient("mysql", "mysql").(*sql.DB)

	gosoAssert.SqlTableHasOneRowOnly(t, db, "mysql_test_models")
	gosoAssert.SqlColumnHasSpecificValue(t, db, "mysql_test_models", "name", "testName")
}
