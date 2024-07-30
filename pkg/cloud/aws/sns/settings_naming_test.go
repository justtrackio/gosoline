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
	config   cfg.GosoConf
	settings sns.TopicNameSettings
}

func (s *GetTopicNameTestSuite) SetupTest() {
	s.config = cfg.New()
	s.settings = sns.TopicNameSettings{
		AppId: cfg.AppId{
			Project:     "justtrack",
			Environment: "test",
			Family:      "gosoline",
			Group:       "group",
			Application: "producer",
		},
		ClientName: "default",
		TopicId:    "event",
	}
}

func (s *GetTopicNameTestSuite) setupConfig(settings map[string]interface{}) {
	err := s.config.Option(cfg.WithConfigMap(settings))
	s.NoError(err, "there should be no error on setting up the config")
}

func (s *GetTopicNameTestSuite) TestDefault() {
	name, err := sns.GetTopicName(s.config, s.settings)
	s.NoError(err)
	s.Equal("justtrack-test-gosoline-group-event", name)
}

func (s *GetTopicNameTestSuite) TestDefaultWithPattern() {
	s.setupConfig(map[string]interface{}{
		"cloud.aws.sns.clients.default.naming.pattern": "{app}-{topicId}",
	})

	name, err := sns.GetTopicName(s.config, s.settings)
	s.NoError(err)
	s.Equal("producer-event", name)
}

func (s *GetTopicNameTestSuite) TestSpecificClientWithPattern() {
	s.settings.ClientName = "specific"
	s.setupConfig(map[string]interface{}{
		"cloud.aws.sns.clients.specific.naming.pattern": "{app}-{topicId}",
	})

	name, err := sns.GetTopicName(s.config, s.settings)
	s.NoError(err)
	s.Equal("producer-event", name)
}
