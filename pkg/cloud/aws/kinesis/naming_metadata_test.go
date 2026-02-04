package kinesis_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/cloud/aws/kinesis"
	"github.com/stretchr/testify/suite"
)

func TestGetMetadataTableNameTestSuite(t *testing.T) {
	suite.Run(t, new(GetMetadataTableNameTestSuite))
}

type GetMetadataTableNameTestSuite struct {
	suite.Suite
	config      cfg.GosoConf
	envProvider cfg.EnvProvider
	settings    *kinesis.Settings
}

func (s *GetMetadataTableNameTestSuite) SetupTest() {
	s.envProvider = cfg.NewMemoryEnvProvider()
	s.config = cfg.NewWithInterfaces(s.envProvider)
	s.settings = &kinesis.Settings{
		AppIdentity: cfg.AppIdentity{
			Name: "producer",
			Env:  "env",
			Tags: cfg.AppTags{
				"project": "justtrack",
				"family":  "gosoline",
				"group":   "grp",
			},
		},
		ClientName: "default",
		StreamName: "event",
	}

	err := s.config.Option(cfg.WithEnvKeyReplacer(cfg.DefaultEnvKeyReplacer))
	s.NoError(err)
}

func (s *GetMetadataTableNameTestSuite) setupConfig(settings map[string]any) {
	err := s.config.Option(cfg.WithConfigMap(settings))
	s.NoError(err, "there should be no error on setting up the config")
}

func (s *GetMetadataTableNameTestSuite) setupConfigEnv(settings map[string]string) {
	for k, v := range settings {
		err := s.envProvider.SetEnv(k, v)
		s.NoError(err, "there should be no error on setting up the config")
	}
}

func (s *GetMetadataTableNameTestSuite) TestDefault() {
	name, err := kinesis.GetMetadataTableName(s.config, s.settings)
	s.NoError(err)
	s.EqualValues("env-kinsumer-metadata", name)
}

func (s *GetMetadataTableNameTestSuite) TestDefaultWithPattern() {
	s.setupConfig(map[string]any{
		"cloud.aws.kinesis.clients.default.naming.metadata_pattern": "{app.name}-metadata",
	})

	name, err := kinesis.GetMetadataTableName(s.config, s.settings)
	s.NoError(err)
	s.EqualValues("producer-metadata", name)
}

func (s *GetMetadataTableNameTestSuite) TestSpecificClientWithPattern() {
	s.settings.ClientName = "specific"
	s.setupConfig(map[string]any{
		"cloud.aws.kinesis.clients.specific.naming.metadata_pattern": "{app.name}-metadata",
	})

	name, err := kinesis.GetMetadataTableName(s.config, s.settings)
	s.NoError(err)
	s.EqualValues("producer-metadata", name)
}

func (s *GetMetadataTableNameTestSuite) TestSpecificClientWithFallbackPattern() {
	s.settings.ClientName = "specific"
	s.setupConfig(map[string]any{
		"cloud.aws.kinesis.clients.default.naming.metadata_pattern": "{app.name}-metadata",
	})

	name, err := kinesis.GetMetadataTableName(s.config, s.settings)
	s.NoError(err)
	s.EqualValues("producer-metadata", name)
}

func (s *GetMetadataTableNameTestSuite) TestSpecificClientWithFallbackPatternViaEnv() {
	s.settings.ClientName = "specific"
	s.setupConfigEnv(map[string]string{
		"CLOUD_AWS_KINESIS_CLIENTS_DEFAULT_NAMING_METADATA_PATTERN": "!nodecode {app.name}-metadata",
	})

	name, err := kinesis.GetMetadataTableName(s.config, s.settings)
	s.NoError(err)
	s.EqualValues("producer-metadata", name)
}
