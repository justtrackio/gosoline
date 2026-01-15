package kafka

import (
	"fmt"
	"strings"

	"github.com/justtrackio/gosoline/pkg/cfg"
)

type KafkaNamingSettings struct {
	TopicPattern string `cfg:"topic_pattern,nodecode" default:"{app.tags.project}-{app.env}-{app.tags.family}-{app.tags.group}-{topicId}"`
	GroupPattern string `cfg:"group_pattern,nodecode" default:"{app.tags.project}-{app.env}-{app.tags.family}-{app.tags.group}-{app.name}-{groupId}"`
}

func NormalizeKafkaName(name string) string {
	return strings.ReplaceAll(name, "_", "-")
}

func BuildFullTopicName(config cfg.Config, identity cfg.AppIdentity, topicId string) (string, error) {
	namingSettings := &KafkaNamingSettings{}
	if err := config.UnmarshalKey("kafka.naming", namingSettings); err != nil {
		return "", fmt.Errorf("failed to unmarshal kafka naming settings for key 'kafka.naming' to build kafka topic name: %w", err)
	}

	// Use NamingTemplate for strict placeholder validation and pattern-driven tag requirements
	tmpl := cfg.NewNamingTemplate(namingSettings.TopicPattern, "topicId")
	tmpl.WithResourceValue("topicId", topicId)

	name, err := tmpl.ValidateAndExpand(identity)
	if err != nil {
		return "", fmt.Errorf("kafka topic naming failed: %w", err)
	}

	return NormalizeKafkaName(name), nil
}

func BuildFullConsumerGroupId(config cfg.Config, groupId string) (string, error) {
	identity, err := cfg.GetAppIdentityFromConfig(config)
	if err != nil {
		return "", fmt.Errorf("failed to get app identity from config: %w", err)
	}

	namingSettings := &KafkaNamingSettings{}
	if err := config.UnmarshalKey("kafka.naming", namingSettings); err != nil {
		return "", fmt.Errorf("failed to unmarshal kafka naming settings for key 'kafka.naming' to build kakfa consumer group id: %w", err)
	}

	// Use NamingTemplate for strict placeholder validation and pattern-driven tag requirements
	tmpl := cfg.NewNamingTemplate(namingSettings.GroupPattern, "groupId")
	tmpl.WithResourceValue("groupId", groupId)

	name, err := tmpl.ValidateAndExpand(identity)
	if err != nil {
		return "", fmt.Errorf("kafka consumer group naming failed: %w", err)
	}

	return NormalizeKafkaName(name), nil
}
