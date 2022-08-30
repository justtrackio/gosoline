package kafka_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kafka"
	"github.com/stretchr/testify/suite"
)

func TestKafkaNamingTestSuite(t *testing.T) {
	suite.Run(t, new(KafkaNamingTestSuite))
}

type KafkaNamingTestSuite struct {
	suite.Suite
	config  cfg.GosoConf
	appID   cfg.AppId
	topicId string
	groupId string
}

func (s *KafkaNamingTestSuite) SetupTest() {
	s.config = cfg.New()
	s.appID = cfg.AppId{
		Project:     "justtrack",
		Environment: "test",
		Family:      "gosoline",
		Group:       "group",
		Application: "producer",
	}
	s.topicId = "topic_a"
	s.groupId = "c-group-1"
}

func (s *KafkaNamingTestSuite) setupConfig(settings map[string]interface{}) {
	err := s.config.Option(cfg.WithConfigMap(settings))
	s.NoError(err, "there should be no error on setting up the config")
}

func (s *KafkaNamingTestSuite) TestDefaultTopicId() {
	s.Equal(kafka.FQTopicName(s.config, s.appID, s.topicId), "test-topic-a")
}

func (s *KafkaNamingTestSuite) TestDefaultGroupId() {
	s.Equal(kafka.FQGroupId(s.config, s.appID, s.groupId), "test-producer-c-group-1")
}

func (s *KafkaNamingTestSuite) TestLegacyGroupId() {
	s.Equal(kafka.FQGroupId(s.config, s.appID, ""), "producer")
}

func (s *KafkaNamingTestSuite) TestTopicIdWithPattern() {
	s.setupConfig(map[string]interface{}{
		"kafka.naming.topic_pattern": "{app}-{topicId}",
	})

	s.Equal(kafka.FQTopicName(s.config, s.appID, s.topicId), "producer-topic-a")
}

func (s *KafkaNamingTestSuite) TestGroupIdWithPattern() {
	s.setupConfig(map[string]interface{}{
		"kafka.naming.group_pattern": "{app}-{groupId}",
	})

	s.Equal(kafka.FQGroupId(s.config, s.appID, s.groupId), "producer-c-group-1")
}
