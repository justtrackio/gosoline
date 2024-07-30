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
	config   cfg.GosoConf
	settings sqs.QueueNameSettings
}

func (s *GetSqsQueueNameTestSuite) SetupTest() {
	s.config = cfg.New()
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
}

func (s *GetSqsQueueNameTestSuite) setupConfig(settings map[string]interface{}) {
	err := s.config.Option(cfg.WithConfigMap(settings))
	s.NoError(err, "there should be no error on setting up the config")
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
	s.setupConfig(map[string]interface{}{
		"cloud.aws.sqs.clients.default.naming.pattern": "{app}-{queueId}",
	})

	name, err := sqs.GetQueueName(s.config, s.settings)
	s.NoError(err)
	s.Equal("producer-event", name)
}

func (s *GetSqsQueueNameTestSuite) TestSpecificClientWithPattern() {
	s.settings.ClientName = "specific"
	s.setupConfig(map[string]interface{}{
		"cloud.aws.sqs.clients.specific.naming.pattern": "{app}-{queueId}",
	})

	name, err := sqs.GetQueueName(s.config, s.settings)
	s.NoError(err)
	s.Equal("producer-event", name)
}
