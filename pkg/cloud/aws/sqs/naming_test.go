package sqs_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/cloud/aws/sqs"
	"github.com/stretchr/testify/suite"
)

func TestGetSqsQueueNameTestSuite(t *testing.T) {
	suite.Run(t, new(GetSqsQueueNameTestSuite))
}

type GetSqsQueueNameTestSuite struct {
	suite.Suite
	config      cfg.GosoConf
	envProvider cfg.EnvProvider
	settings    sqs.QueueNameSettings
}

func (s *GetSqsQueueNameTestSuite) SetupTest() {
	s.envProvider = cfg.NewMemoryEnvProvider()
	s.config = cfg.NewWithInterfaces(s.envProvider)
	s.settings = sqs.QueueNameSettings{
		Identity: cfg.Identity{
			Name:      "producer",
			Env:       "test",
			Namespace: "{app.tags.project}.{app.env}.{app.tags.family}.{app.tags.group}",
			Tags: cfg.Tags{
				"project": "justtrack",
				"family":  "gosoline",
				"group":   "group",
			},
		},
		ClientName: "default",
		QueueId:    "event",
	}

	err := s.config.Option(cfg.WithEnvKeyReplacer(cfg.DefaultEnvKeyReplacer))
	s.NoError(err)
}

func (s *GetSqsQueueNameTestSuite) setupConfig(settings map[string]any) {
	err := s.config.Option(cfg.WithConfigMap(settings))
	s.NoError(err, "there should be no error on setting up the config")
}

func (s *GetSqsQueueNameTestSuite) setupConfigEnv(settings map[string]string) {
	for k, v := range settings {
		err := s.envProvider.SetEnv(k, v)
		s.NoError(err, "there should be no error on setting up the config")
	}
}

func (s *GetSqsQueueNameTestSuite) TestDefault() {
	name, err := sqs.GetQueueName(s.config, s.settings)
	s.NoError(err)
	s.Equal("justtrack-test-gosoline-group-event", name)
}

func (s *GetSqsQueueNameTestSuite) TestDefaultFifo() {
	s.settings.FifoEnabled = true

	name, err := sqs.GetQueueName(s.config, s.settings)
	s.NoError(err)
	s.Equal("justtrack-test-gosoline-group-event.fifo", name)
}

func (s *GetSqsQueueNameTestSuite) TestDefaultWithPattern() {
	s.setupConfig(map[string]any{
		"cloud.aws.sqs.clients.default.naming.queue_pattern": "{app.name}-{queueId}",
	})

	name, err := sqs.GetQueueName(s.config, s.settings)
	s.NoError(err)
	s.Equal("producer-event", name)
}

func (s *GetSqsQueueNameTestSuite) TestSpecificClientWithPattern() {
	s.settings.ClientName = "specific"
	s.setupConfig(map[string]any{
		"cloud.aws.sqs.clients.specific.naming.queue_pattern": "{app.name}-{queueId}",
	})

	name, err := sqs.GetQueueName(s.config, s.settings)
	s.NoError(err)
	s.Equal("producer-event", name)
}

func (s *GetSqsQueueNameTestSuite) TestSpecificClientWithFallbackPattern() {
	s.settings.ClientName = "specific"
	s.setupConfig(map[string]any{
		"cloud.aws.sqs.clients.default.naming.queue_pattern": "{app.name}-{queueId}",
	})

	name, err := sqs.GetQueueName(s.config, s.settings)
	s.NoError(err)
	s.Equal("producer-event", name)
}

func (s *GetSqsQueueNameTestSuite) TestSpecificClientWithFallbackPatternViaEnv() {
	s.settings.ClientName = "specific"
	s.setupConfigEnv(map[string]string{
		"CLOUD_AWS_SQS_CLIENTS_DEFAULT_NAMING_QUEUE_PATTERN": "!nodecode {app.name}-{queueId}",
	})

	name, err := sqs.GetQueueName(s.config, s.settings)
	s.NoError(err)
	s.Equal("producer-event", name)
}

func (s *GetSqsQueueNameTestSuite) TestUnknownPlaceholderReturnsError() {
	s.setupConfig(map[string]any{
		"cloud.aws.sqs.clients.default.naming.queue_pattern": "{project}-{queueId}",
	})

	_, err := sqs.GetQueueName(s.config, s.settings)
	s.Error(err)
	s.Contains(err.Error(), "unknown placeholder {project}")
}

func (s *GetSqsQueueNameTestSuite) TestMissingTagsOnlyFailsIfPatternRequiresThem() {
	// QueuePattern doesn't use tags, so missing tags should not cause error
	s.settings.Identity.Tags = nil
	s.settings.Identity.Namespace = "{app.env}"
	s.setupConfig(map[string]any{
		"cloud.aws.sqs.clients.default.naming.queue_pattern": "{app.env}-{queueId}",
	})

	name, err := sqs.GetQueueName(s.config, s.settings)
	s.NoError(err)
	s.Equal("test-event", name)
}

func (s *GetSqsQueueNameTestSuite) TestMissingRequiredTagReturnsError() {
	// QueuePattern uses project tag but it's missing
	s.settings.Identity.Tags = cfg.Tags{}
	s.setupConfig(map[string]any{
		"cloud.aws.sqs.clients.default.naming.queue_pattern": "{app.tags.project}-{queueId}",
	})

	_, err := sqs.GetQueueName(s.config, s.settings)
	s.Error(err)
	s.Contains(err.Error(), "unknown placeholder {app.tags.project}")
}

func (s *GetSqsQueueNameTestSuite) TestCustomDelimiter() {
	s.setupConfig(map[string]any{
		"cloud.aws.sqs.clients.default.naming.queue_pattern":   "{app.namespace}.{queueId}",
		"cloud.aws.sqs.clients.default.naming.queue_delimiter": ".",
	})

	name, err := sqs.GetQueueName(s.config, s.settings)
	s.NoError(err)
	s.Equal("justtrack.test.gosoline.group.event", name)
}
