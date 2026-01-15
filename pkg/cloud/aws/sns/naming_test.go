package sns_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/cloud/aws/sns"
	"github.com/stretchr/testify/suite"
)

func TestGetTopicNameTestSuite(t *testing.T) {
	suite.Run(t, new(GetTopicNameTestSuite))
}

type GetTopicNameTestSuite struct {
	suite.Suite
	config      cfg.GosoConf
	envProvider cfg.EnvProvider
	settings    sns.TopicNameSettings
}

func (s *GetTopicNameTestSuite) SetupTest() {
	s.envProvider = cfg.NewMemoryEnvProvider()
	s.config = cfg.NewWithInterfaces(s.envProvider)
	s.settings = sns.TopicNameSettings{
		Identity: cfg.Identity{
			Name:      "producer",
			Env:       "test",
			Namespace: "{app.tags.project}.{app.env}.{app.tags.family}.{app.tags.group}",
			Tags: map[string]string{
				"project": "justtrack",
				"family":  "gosoline",
				"group":   "group",
			},
		},
		ClientName: "default",
		TopicId:    "event",
	}

	err := s.config.Option(cfg.WithEnvKeyReplacer(cfg.DefaultEnvKeyReplacer))
	s.NoError(err)
}

func (s *GetTopicNameTestSuite) setupConfig(settings map[string]any) {
	err := s.config.Option(cfg.WithConfigMap(settings))
	s.NoError(err, "there should be no error on setting up the config")
}

func (s *GetTopicNameTestSuite) TestDefault() {
	name, err := sns.GetTopicName(s.config, s.settings)
	s.NoError(err)
	s.Equal("justtrack-test-gosoline-group-event", name)
}

func (s *GetTopicNameTestSuite) setupConfigEnv(settings map[string]string) {
	for k, v := range settings {
		err := s.envProvider.SetEnv(k, v)
		s.NoError(err, "there should be no error on setting up the config")
	}
}

func (s *GetTopicNameTestSuite) TestDefaultWithPattern() {
	s.setupConfig(map[string]any{
		"cloud.aws.sns.clients.default.naming.topic_pattern": "{app.name}-{topicId}",
	})

	name, err := sns.GetTopicName(s.config, s.settings)
	s.NoError(err)
	s.Equal("producer-event", name)
}

func (s *GetTopicNameTestSuite) TestSpecificClientWithPattern() {
	s.settings.ClientName = "specific"
	s.setupConfig(map[string]any{
		"cloud.aws.sns.clients.specific.naming.topic_pattern": "{app.name}-{topicId}",
	})

	name, err := sns.GetTopicName(s.config, s.settings)
	s.NoError(err)
	s.Equal("producer-event", name)
}

func (s *GetTopicNameTestSuite) TestSpecificClientWithFallbackPattern() {
	s.settings.ClientName = "specific"
	s.setupConfig(map[string]any{
		"cloud.aws.sns.clients.default.naming.topic_pattern": "{app.name}-{topicId}",
	})

	name, err := sns.GetTopicName(s.config, s.settings)
	s.NoError(err)
	s.Equal("producer-event", name)
}

func (s *GetTopicNameTestSuite) TestSpecificClientWithFallbackPatternViaEnv() {
	s.settings.ClientName = "specific"
	s.setupConfigEnv(map[string]string{
		"CLOUD_AWS_SNS_CLIENTS_SPECIFIC_NAMING_TOPIC_PATTERN": "!nodecode {app.name}-{topicId}",
	})

	name, err := sns.GetTopicName(s.config, s.settings)
	s.NoError(err)
	s.Equal("producer-event", name)
}

func (s *GetTopicNameTestSuite) TestUnknownPlaceholderReturnsError() {
	s.setupConfig(map[string]any{
		"cloud.aws.sns.clients.default.naming.topic_pattern": "{project}-{topicId}",
	})

	_, err := sns.GetTopicName(s.config, s.settings)
	s.Error(err)
	s.Contains(err.Error(), "unknown placeholder {project}")
}

func (s *GetTopicNameTestSuite) TestMissingTagsOnlyFailsIfPatternRequiresThem() {
	// TopicPattern doesn't use tags, so missing tags should not cause error
	s.settings.Identity.Tags = nil
	s.settings.Identity.Namespace = "{app.env}"
	s.setupConfig(map[string]any{
		"cloud.aws.sns.clients.default.naming.topic_pattern": "{app.env}-{topicId}",
	})

	name, err := sns.GetTopicName(s.config, s.settings)
	s.NoError(err)
	s.Equal("test-event", name)
}

func (s *GetTopicNameTestSuite) TestCustomDelimiter() {
	s.setupConfig(map[string]any{
		"cloud.aws.sns.clients.default.naming.topic_pattern":   "{app.namespace}.{topicId}",
		"cloud.aws.sns.clients.default.naming.topic_delimiter": ".",
	})

	name, err := sns.GetTopicName(s.config, s.settings)
	s.NoError(err)
	s.Equal("justtrack.test.gosoline.group.event", name)
}
