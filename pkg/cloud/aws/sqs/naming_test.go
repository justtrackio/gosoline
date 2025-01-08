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
		AppId: cfg.AppId{
			Project:     "justtrack",
			Environment: "test",
			Family:      "gosoline",
			Group:       "group",
			Application: "producer",
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
		"cloud.aws.sqs.clients.default.naming.pattern": "{app}-{queueId}",
	})

	name, err := sqs.GetQueueName(s.config, s.settings)
	s.NoError(err)
	s.Equal("producer-event", name)
}

func (s *GetSqsQueueNameTestSuite) TestSpecificClientWithPattern() {
	s.settings.ClientName = "specific"
	s.setupConfig(map[string]any{
		"cloud.aws.sqs.clients.specific.naming.pattern": "{app}-{queueId}",
	})

	name, err := sqs.GetQueueName(s.config, s.settings)
	s.NoError(err)
	s.Equal("producer-event", name)
}

func (s *GetSqsQueueNameTestSuite) TestSpecificClientWithFallbackPattern() {
	s.settings.ClientName = "specific"
	s.setupConfig(map[string]any{
		"cloud.aws.sqs.clients.default.naming.pattern": "{app}-{queueId}",
	})

	name, err := sqs.GetQueueName(s.config, s.settings)
	s.NoError(err)
	s.Equal("producer-event", name)
}

func (s *GetSqsQueueNameTestSuite) TestSpecificClientWithFallbackPatternViaEnv() {
	s.settings.ClientName = "specific"
	s.setupConfigEnv(map[string]string{
		"CLOUD_AWS_SQS_CLIENTS_DEFAULT_NAMING_PATTERN": "!nodecode {app}-{queueId}",
	})

	name, err := sqs.GetQueueName(s.config, s.settings)
	s.NoError(err)
	s.Equal("producer-event", name)
}
