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
		Identity: cfg.Identity{
			Name:      "producer",
			Env:       "env",
			Namespace: "{app.tags.project}.{app.env}.{app.tags.family}",
			Tags: cfg.Tags{
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

	// Ensure namespaceParts are initialized
	err = s.settings.Identity.PadFromConfig(s.config)
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
	s.EqualValues("justtrack-env-gosoline-kinsumer-metadata", name)
}

func (s *GetMetadataTableNameTestSuite) TestDefaultWithPattern() {
	s.setupConfig(map[string]any{
		"cloud.aws.kinesis.clients.default.naming.metadata_table_pattern": "{app.name}-metadata",
	})

	name, err := kinesis.GetMetadataTableName(s.config, s.settings)
	s.NoError(err)
	s.EqualValues("producer-metadata", name)
}

func (s *GetMetadataTableNameTestSuite) TestSpecificClientWithPattern() {
	s.settings.ClientName = "specific"
	s.setupConfig(map[string]any{
		"cloud.aws.kinesis.clients.specific.naming.metadata_table_pattern": "{app.name}-metadata",
	})

	name, err := kinesis.GetMetadataTableName(s.config, s.settings)
	s.NoError(err)
	s.EqualValues("producer-metadata", name)
}

func (s *GetMetadataTableNameTestSuite) TestSpecificClientWithFallbackPattern() {
	s.settings.ClientName = "specific"
	s.setupConfig(map[string]any{
		"cloud.aws.kinesis.clients.default.naming.metadata_table_pattern": "{app.name}-metadata",
	})

	name, err := kinesis.GetMetadataTableName(s.config, s.settings)
	s.NoError(err)
	s.EqualValues("producer-metadata", name)
}

func (s *GetMetadataTableNameTestSuite) TestSpecificClientWithFallbackPatternViaEnv() {
	s.settings.ClientName = "specific"
	s.setupConfigEnv(map[string]string{
		"CLOUD_AWS_KINESIS_CLIENTS_DEFAULT_NAMING_METADATA_TABLE_PATTERN": "!nodecode {app.name}-metadata",
	})

	name, err := kinesis.GetMetadataTableName(s.config, s.settings)
	s.NoError(err)
	s.EqualValues("producer-metadata", name)
}

func (s *GetMetadataTableNameTestSuite) TestCustomDelimiter() {
	s.setupConfig(map[string]any{
		"cloud.aws.kinesis.clients.default.naming.metadata_table_pattern":   "{app.namespace}.{app.name}",
		"cloud.aws.kinesis.clients.default.naming.metadata_table_delimiter": ".",
	})

	name, err := kinesis.GetMetadataTableName(s.config, s.settings)
	s.NoError(err)
	s.EqualValues("justtrack.env.gosoline.producer", name)
}

func (s *GetMetadataTableNameTestSuite) TestDelimiterFallback() {
	s.settings.ClientName = "specific"
	s.setupConfig(map[string]any{
		"cloud.aws.kinesis.clients.default.naming.metadata_table_delimiter": "_",
	})

	name, err := kinesis.GetMetadataTableName(s.config, s.settings)
	s.NoError(err)
	// Delimiter only affects namespace parts, not literal separators in pattern
	s.EqualValues("justtrack_env_gosoline-kinsumer-metadata", name)
}
