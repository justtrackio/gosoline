package kinesis_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/cloud/aws/kinesis"
	"github.com/stretchr/testify/suite"
)

func TestGetStreamNameTestSuite(t *testing.T) {
	suite.Run(t, new(GetStreamNameTestSuite))
}

type GetStreamNameTestSuite struct {
	suite.Suite
	config      cfg.GosoConf
	envProvider cfg.EnvProvider
	settings    *kinesis.Settings
}

func (s *GetStreamNameTestSuite) SetupTest() {
	s.envProvider = cfg.NewMemoryEnvProvider()
	s.config = cfg.NewWithInterfaces(s.envProvider)
	s.settings = &kinesis.Settings{
		Identity: cfg.Identity{
			Name:      "producer",
			Env:       "env",
			Namespace: "{app.tags.project}.{app.env}.{app.tags.family}.{app.tags.group}",
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

func (s *GetStreamNameTestSuite) setupConfig(settings map[string]any) {
	err := s.config.Option(cfg.WithConfigMap(settings))
	s.NoError(err, "there should be no error on setting up the config")
}

func (s *GetStreamNameTestSuite) setupConfigEnv(settings map[string]string) {
	for k, v := range settings {
		err := s.envProvider.SetEnv(k, v)
		s.NoError(err, "there should be no error on setting up the config")
	}
}

func (s *GetStreamNameTestSuite) TestDefault() {
	name, err := kinesis.GetStreamName(s.config, s.settings)
	s.NoError(err)
	s.EqualValues("justtrack-env-gosoline-grp-event", string(name))
}

func (s *GetStreamNameTestSuite) TestDefaultWithPattern() {
	s.setupConfig(map[string]any{
		"cloud.aws.kinesis.clients.default.naming.stream_pattern": "{app.name}-{streamName}",
	})

	name, err := kinesis.GetStreamName(s.config, s.settings)
	s.NoError(err)
	s.EqualValues("producer-event", name)
}

func (s *GetStreamNameTestSuite) TestSpecificClientWithPattern() {
	s.settings.ClientName = "specific"
	s.setupConfig(map[string]any{
		"cloud.aws.kinesis.clients.specific.naming.stream_pattern": "{app.name}-{streamName}",
	})

	name, err := kinesis.GetStreamName(s.config, s.settings)
	s.NoError(err)
	s.EqualValues("producer-event", name)
}

func (s *GetStreamNameTestSuite) TestSpecificClientWithFallbackPattern() {
	s.settings.ClientName = "specific"
	s.setupConfig(map[string]any{
		"cloud.aws.kinesis.clients.default.naming.stream_pattern": "{app.name}-{streamName}",
	})

	name, err := kinesis.GetStreamName(s.config, s.settings)
	s.NoError(err)
	s.EqualValues("producer-event", name)
}

func (s *GetStreamNameTestSuite) TestSpecificClientWithFallbackPatternViaEnv() {
	s.settings.ClientName = "specific"
	s.setupConfigEnv(map[string]string{
		"CLOUD_AWS_KINESIS_CLIENTS_DEFAULT_NAMING_STREAM_PATTERN": "!nodecode {app.name}-{streamName}",
	})

	name, err := kinesis.GetStreamName(s.config, s.settings)
	s.NoError(err)
	s.EqualValues("producer-event", name)
}

func (s *GetStreamNameTestSuite) TestUnknownPlaceholderReturnsError() {
	s.setupConfig(map[string]any{
		"cloud.aws.kinesis.clients.default.naming.stream_pattern": "{project}-{streamName}",
	})

	_, err := kinesis.GetStreamName(s.config, s.settings)
	s.Error(err)
	s.Contains(err.Error(), "unknown placeholder {project} in pattern")
}

func (s *GetStreamNameTestSuite) TestMissingTagsOnlyFailsIfPatternRequiresThem() {
	// StreamPattern doesn't use tags, so missing tags should not cause error
	s.settings.Identity.Tags = nil
	s.settings.Identity.Namespace = "{app.env}"
	s.setupConfig(map[string]any{
		"cloud.aws.kinesis.clients.default.naming.stream_pattern": "{app.env}-{streamName}",
	})

	// Re-initialize namespaceParts with the new namespace
	err := s.settings.Identity.PadFromConfig(s.config)
	s.NoError(err)

	name, err := kinesis.GetStreamName(s.config, s.settings)
	s.NoError(err)
	s.EqualValues("env-event", name)
}

func (s *GetStreamNameTestSuite) TestCustomDelimiter() {
	s.setupConfig(map[string]any{
		"cloud.aws.kinesis.clients.default.naming.stream_pattern":   "{app.namespace}.{streamName}",
		"cloud.aws.kinesis.clients.default.naming.stream_delimiter": ".",
	})

	name, err := kinesis.GetStreamName(s.config, s.settings)
	s.NoError(err)
	s.EqualValues("justtrack.env.gosoline.grp.event", name)
}

func (s *GetStreamNameTestSuite) TestDelimiterFallback() {
	s.settings.ClientName = "specific"
	s.setupConfig(map[string]any{
		"cloud.aws.kinesis.clients.default.naming.stream_delimiter": "_",
	})

	name, err := kinesis.GetStreamName(s.config, s.settings)
	s.NoError(err)
	// Delimiter only affects namespace parts, not literal separators in pattern
	s.EqualValues("justtrack_env_gosoline_grp-event", name)
}
