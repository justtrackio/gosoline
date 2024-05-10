package ddb_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/ddb"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/stretchr/testify/suite"
)

func TestTableNameTestSuite(t *testing.T) {
	suite.Run(t, new(TableNameTestSuite))
}

type TableNameTestSuite struct {
	suite.Suite
	config      cfg.GosoConf
	envProvider cfg.EnvProvider
	settings    *ddb.Settings
}

func (s *TableNameTestSuite) SetupTest() {
	s.envProvider = cfg.NewMemoryEnvProvider()
	s.config = cfg.NewWithInterfaces(s.envProvider)
	s.settings = &ddb.Settings{
		ModelId: mdl.ModelId{
			Project:     "justtrack",
			Environment: "test",
			Family:      "gosoline",
			Group:       "group",
			Application: "producer",
			Name:        "event",
		},
		ClientName: "default",
	}

	err := s.config.Option(cfg.WithEnvKeyReplacer(cfg.DefaultEnvKeyReplacer))
	s.NoError(err)
}

func (s *TableNameTestSuite) setupConfig(settings map[string]any) {
	err := s.config.Option(cfg.WithConfigMap(settings))
	s.NoError(err, "there should be no error on setting up the config")
}

func (s *TableNameTestSuite) setupConfigEnv(settings map[string]string) {
	for k, v := range settings {
		err := s.envProvider.SetEnv(k, v)
		s.NoError(err, "there should be no error on setting up the config")
	}
}

func (s *TableNameTestSuite) TestDefault() {
	name, err := ddb.TableName(s.config, s.settings)
	if err != nil {
		s.FailNow("there should be no error on getting the table name", err)
	}

	s.Equal("justtrack-test-gosoline-group-event", name)
}

func (s *TableNameTestSuite) TestDefaultWithPattern() {
	s.setupConfig(map[string]any{
		"cloud.aws.dynamodb.clients.default.naming.pattern": "{app}-{modelId}",
	})

	name, err := ddb.TableName(s.config, s.settings)
	if err != nil {
		s.FailNow("there should be no error on getting the table name", err)
	}

	s.Equal("producer-event", name)
}

func (s *TableNameTestSuite) TestSpecificClientWithPattern() {
	s.settings.ClientName = "specific"
	s.setupConfig(map[string]any{
		"cloud.aws.dynamodb.clients.specific.naming.pattern": "{app}-{modelId}",
	})

	name, err := ddb.TableName(s.config, s.settings)
	if err != nil {
		s.FailNow("there should be no error on getting the table name", err)
	}

	s.Equal("producer-event", name)
}

func (s *TableNameTestSuite) TestPatternFromTableSettings() {
	s.settings.TableNamingSettings = ddb.TableNamingSettings{
		Pattern: "this-is-an-fqn-overwrite",
	}

	name, err := ddb.TableName(s.config, s.settings)
	if err != nil {
		s.FailNow("there should be no error on getting the table name", err)
	}

	s.Equal("this-is-an-fqn-overwrite", name)
}

func (s *TableNameTestSuite) TestSpecificClientWithFallbackPattern() {
	s.settings.ClientName = "specific"
	s.setupConfig(map[string]any{
		"cloud.aws.dynamodb.clients.default.naming.pattern": "{app}-{modelId}",
	})

	name, err := ddb.TableName(s.config, s.settings)
	if err != nil {
		s.FailNow("there should be no error on getting the table name", err)
	}

	s.Equal("producer-event", name)
}

func (s *TableNameTestSuite) TestSpecificClientWithFallbackPatternViaEnv() {
	s.settings.ClientName = "specific"
	s.setupConfigEnv(map[string]string{
		"CLOUD_AWS_DYNAMODB_CLIENTS_DEFAULT_NAMING_PATTERN": "!nodecode {app}-{modelId}",
	})

	name, err := ddb.TableName(s.config, s.settings)
	if err != nil {
		s.FailNow("there should be no error on getting the table name", err)
	}

	s.Equal("producer-event", name)
}
