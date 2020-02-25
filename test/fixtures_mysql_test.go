//+build integration

package test_test

import (
	"context"
	"github.com/applike/gosoline/pkg/cfg"
	db_repo "github.com/applike/gosoline/pkg/db-repo"
	"github.com/applike/gosoline/pkg/fixtures"
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/test"
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
			Writer:  fixtures.MySqlFixtureWriterFactory(&TestModelMetadata),
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

	test.Boot(configFile)
	defer test.Shutdown()

	loader := fixtures.NewFixtureLoader(mysqlTestFixtures())
	logger := mon.NewLogger()
	config := configFromFiles("test_configs/config.mysql.test.yml", "test_configs/config.fixtures_mysql.test.yml")

	err := loader.Boot(config, logger)
	assert.NoError(t, err)

	settings := db_repo.Settings{
		AppId:    cfg.GetAppIdFromConfig(config),
		Metadata: TestModelMetadata,
	}

	repo := db_repo.New(config, logger, settings)

	result := MysqlTestModel{}
	_ = repo.Read(context.Background(), mdl.Uint(1), &result)

	assert.NoError(t, err)
	assert.Equal(t, "testName", *result.Name)
}
