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
	config   cfg.GosoConf
	settings *ddb.Settings
}

func (s *TableNameTestSuite) SetupTest() {
	s.config = cfg.New()
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
}

func (s *TableNameTestSuite) setupConfig(settings map[string]interface{}) {
	err := s.config.Option(cfg.WithConfigMap(settings))
	s.NoError(err, "there should be no error on setting up the config")
}

func (s *TableNameTestSuite) TestDefault() {
	name := ddb.TableName(s.config, s.settings)
	s.Equal("justtrack-test-gosoline-group-event", name)
}

func (s *TableNameTestSuite) TestDefaultWithPattern() {
	s.setupConfig(map[string]interface{}{
		"cloud.aws.dynamodb.clients.default.naming.pattern": "{app}-{modelId}",
	})

	name := ddb.TableName(s.config, s.settings)
	s.Equal("producer-event", name)
}

func (s *TableNameTestSuite) TestSpecificClientWithPattern() {
	s.settings.ClientName = "specific"
	s.setupConfig(map[string]interface{}{
		"cloud.aws.dynamodb.clients.specific.naming.pattern": "{app}-{modelId}",
	})

	name := ddb.TableName(s.config, s.settings)
	s.Equal("producer-event", name)
}

func (s *TableNameTestSuite) TestPatternFromTableSettings() {
	s.settings.TableNamingSettings = ddb.TableNamingSettings{
		Pattern: "this-is-an-fqn-overwrite",
	}

	name := ddb.TableName(s.config, s.settings)
	s.Equal("this-is-an-fqn-overwrite", name)
}
