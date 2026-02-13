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
	config   cfg.GosoConf
	identity cfg.Identity
	topicId  string
	groupId  string
}

func (s *KafkaNamingTestSuite) SetupTest() {
	s.config = cfg.New(map[string]any{
		"app": map[string]any{
			"env":       "env",
			"name":      "appname",
			"namespace": "{app.tags.project}.{app.env}.{app.tags.family}.{app.tags.group}",
			"tags": map[string]any{
				"project": "project",
				"family":  "family",
				"group":   "group",
			},
		},
	})
	s.identity = cfg.Identity{
		Name:      "producer",
		Env:       "test",
		Namespace: "{app.tags.project}.{app.env}.{app.tags.family}.{app.tags.group}",
		Tags: cfg.Tags{
			"project": "justtrack",
			"family":  "gosoline",
			"group":   "group",
		},
	}
	s.topicId = "topic_a"
	s.groupId = "c-group-1"
}

func (s *KafkaNamingTestSuite) setupConfig(settings map[string]any) {
	err := s.config.Option(cfg.WithConfigMap(settings))
	s.NoError(err, "there should be no error on setting up the config")
}

func (s *KafkaNamingTestSuite) TestDefaultTopicId() {
	topic, err := kafka.BuildFullTopicName(s.config, s.identity, s.topicId)
	s.NoError(err, "there should be no error")
	s.Equal("justtrack-test-gosoline-group-topic-a", topic)
}

func (s *KafkaNamingTestSuite) TestDefaultGroupId() {
	group, err := kafka.BuildFullConsumerGroupId(s.config, s.groupId)
	s.NoError(err, "there should be no error")
	s.Equal("project-env-family-group-appname-c-group-1", group)
}

func (s *KafkaNamingTestSuite) TestTopicIdWithPattern() {
	s.setupConfig(map[string]any{
		"kafka.naming.topic_pattern": "{app.name}-{topicId}",
	})

	topic, err := kafka.BuildFullTopicName(s.config, s.identity, s.topicId)
	s.NoError(err, "there should be no error")
	s.Equal("producer-topic-a", topic)
}

func (s *KafkaNamingTestSuite) TestGroupIdWithPattern() {
	s.setupConfig(map[string]any{
		"kafka.naming.group_pattern": "{app.name}-{groupId}",
	})

	group, err := kafka.BuildFullConsumerGroupId(s.config, s.groupId)
	s.NoError(err, "there should be no error")
	s.Equal("appname-c-group-1", group)
}

func (s *KafkaNamingTestSuite) TestUnknownPlaceholderReturnsError() {
	s.setupConfig(map[string]any{
		"kafka.naming.topic_pattern": "{project}-{topicId}",
	})

	_, err := kafka.BuildFullTopicName(s.config, s.identity, s.topicId)
	s.Error(err)
	s.Contains(err.Error(), "unknown placeholder {project}")
}

func (s *KafkaNamingTestSuite) TestMissingTagsOnlyFailsIfPatternRequiresThem() {
	// Pattern doesn't use tags, so missing tags should not cause error
	s.identity.Tags = nil
	s.identity.Namespace = "{app.env}"
	s.setupConfig(map[string]any{
		"kafka.naming.topic_pattern": "{app.env}-{topicId}",
	})

	topic, err := kafka.BuildFullTopicName(s.config, s.identity, s.topicId)
	s.NoError(err)
	s.Equal("test-topic-a", topic)
}
