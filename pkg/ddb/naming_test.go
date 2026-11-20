package ddb_test

import (
	"fmt"
	"strings"
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
			Name:        "event",
			Env:         "test",
			Application: "producer",
			Tags: map[string]string{
				"project": "justtrack",
				"family":  "gosoline",
				"group":   "group",
			},
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
	s.setupConfig(map[string]any{
		"app.namespace": "{app.tags.project}.{app.env}.{app.tags.family}.{app.tags.group}",
	})

	name, err := ddb.GetTableName(s.config, s.settings)
	if err != nil {
		s.FailNow("there should be no error on getting the table name", err)
	}

	s.Equal("justtrack-test-gosoline-group-event", name)
}

func (s *TableNameTestSuite) TestDefaultWithPattern() {
	s.setupConfig(map[string]any{
		"cloud.aws.dynamodb.clients.default.naming.table_pattern": "{app.name}-{name}",
	})

	name, err := ddb.GetTableName(s.config, s.settings)
	if err != nil {
		s.FailNow("there should be no error on getting the table name", err)
	}

	s.Equal("producer-event", name)
}

func (s *TableNameTestSuite) TestSpecificClientWithPattern() {
	s.settings.ClientName = "specific"
	s.setupConfig(map[string]any{
		"cloud.aws.dynamodb.clients.specific.naming.table_pattern": "{app.name}-{name}",
	})

	name, err := ddb.GetTableName(s.config, s.settings)
	if err != nil {
		s.FailNow("there should be no error on getting the table name", err)
	}

	s.Equal("producer-event", name)
}

func (s *TableNameTestSuite) TestPatternFromTableSettings() {
	s.settings.TableNamingSettings = ddb.TableNamingSettings{
		TablePattern: "this-is-an-fqn-overwrite-event",
	}

	name, err := ddb.GetTableName(s.config, s.settings)
	if err != nil {
		s.FailNow("there should be no error on getting the table name", err)
	}

	s.Equal("this-is-an-fqn-overwrite-event", name)
}

func (s *TableNameTestSuite) TestSpecificClientWithFallbackPattern() {
	s.settings.ClientName = "specific"
	s.setupConfig(map[string]any{
		"cloud.aws.dynamodb.clients.default.naming.table_pattern": "{app.name}-{name}",
	})

	name, err := ddb.GetTableName(s.config, s.settings)
	if err != nil {
		s.FailNow("there should be no error on getting the table name", err)
	}

	s.Equal("producer-event", name)
}

func (s *TableNameTestSuite) TestSpecificClientWithFallbackPatternViaEnv() {
	s.settings.ClientName = "specific"
	s.setupConfigEnv(map[string]string{
		"CLOUD_AWS_DYNAMODB_CLIENTS_DEFAULT_NAMING_TABLE_PATTERN": "!nodecode {app.name}-{name}",
	})

	name, err := ddb.GetTableName(s.config, s.settings)
	if err != nil {
		s.FailNow("there should be no error on getting the table name", err)
	}

	s.Equal("producer-event", name)
}

func (s *TableNameTestSuite) TestTableNameLengthLimit() {
	s.setupConfig(map[string]any{
		"cloud.aws.dynamodb.clients.default.naming.table_pattern": "{app.name}-{name}",
	})

	for _, length := range []int{254, 255, 256, 1000} {
		s.Run(fmt.Sprint(length), func() {
			s.settings.ModelId.Name = strings.Repeat("a", length-len("producer-"))
			original := "producer-" + s.settings.ModelId.Name
			name, err := ddb.GetTableName(s.config, s.settings)
			s.Require().NoError(err)
			if length <= 255 {
				s.Equal(original, name)

				return
			}
			s.Len(name, 255)
			s.True(strings.HasPrefix(name, "producer-"))
			repeated, err := ddb.GetTableName(s.config, s.settings)
			s.Require().NoError(err)
			s.Equal(name, repeated)
		})
	}
}

func (s *TableNameTestSuite) TestLongTableNamesRemainDistinct() {
	s.setupConfig(map[string]any{
		"cloud.aws.dynamodb.clients.default.naming.table_pattern": "{name}",
	})
	s.settings.ModelId.Name = strings.Repeat("a", 255) + "a"
	first, err := ddb.GetTableName(s.config, s.settings)
	s.Require().NoError(err)
	s.settings.ModelId.Name = strings.Repeat("a", 255) + "b"
	second, err := ddb.GetTableName(s.config, s.settings)
	s.Require().NoError(err)
	s.Len(first, 255)
	s.Len(second, 255)
	s.NotEqual(first, second)
}

func (s *TableNameTestSuite) TestInvalidPattern() {
	s.setupConfig(map[string]any{
		"cloud.aws.dynamodb.clients.default.naming.table_pattern": "{unknown}",
	})
	name, err := ddb.GetTableName(s.config, s.settings)
	s.Error(err)
	s.Empty(name)
}
