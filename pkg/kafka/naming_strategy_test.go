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

func (s *KafkaNamingTestSuite) setupConfig(settings map[string]any) {
	err := s.config.Option(cfg.WithConfigMap(settings))
	s.NoError(err, "there should be no error on setting up the config")
}

func (s *KafkaNamingTestSuite) TestDefaultTopicId() {
	topic, err := kafka.FQTopicName(s.config, s.appID, s.topicId)
	s.NoError(err, "there should be no error")
	s.Equal(topic, "test-topic-a")
}

func (s *KafkaNamingTestSuite) TestDefaultGroupId() {
	group, err := kafka.FQGroupId(s.config, s.appID, s.groupId)
	s.NoError(err, "there should be no error")
	s.Equal(group, "test-producer-c-group-1")
}

func (s *KafkaNamingTestSuite) TestLegacyGroupId() {
	group, err := kafka.FQGroupId(s.config, s.appID, "")
	s.NoError(err, "there should be no error")
	s.Equal(group, "producer")
}

func (s *KafkaNamingTestSuite) TestTopicIdWithPattern() {
	s.setupConfig(map[string]any{
		"kafka.naming.topic_pattern": "{app}-{topicId}",
	})

	topic, err := kafka.FQTopicName(s.config, s.appID, s.topicId)
	s.NoError(err, "there should be no error")
	s.Equal(topic, "producer-topic-a")
}

func (s *KafkaNamingTestSuite) TestGroupIdWithPattern() {
	s.setupConfig(map[string]any{
		"kafka.naming.group_pattern": "{app}-{groupId}",
	})

	group, err := kafka.FQGroupId(s.config, s.appID, s.groupId)
	s.NoError(err, "there should be no error")
	s.Equal(group, "producer-c-group-1")
}
