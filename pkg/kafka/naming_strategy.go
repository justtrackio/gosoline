package kafka

import (
	"fmt"
	"strings"

	"github.com/justtrackio/gosoline/pkg/cfg"
)

type KafkaNamingSettings struct {
	TopicPattern   string `cfg:"topic_pattern,nodecode" default:"{app.namespace}-{topicId}"`
	TopicDelimiter string `cfg:"topic_delimiter" default:"-"`
	GroupPattern   string `cfg:"group_pattern,nodecode" default:"{app.namespace}-{app.name}-{groupId}"`
	GroupDelimiter string `cfg:"group_delimiter" default:"-"`
}

func NormalizeKafkaName(name string) string {
	return strings.ReplaceAll(name, "_", "-")
}

func BuildFullTopicName(config cfg.Config, identity cfg.Identity, topicId string) (string, error) {
	namingSettings := &KafkaNamingSettings{}
	if err := config.UnmarshalKey("kafka.naming", namingSettings); err != nil {
		return "", fmt.Errorf("failed to unmarshal kafka naming settings for key 'kafka.naming' to build kafka topic name: %w", err)
	}

	if err := identity.PadFromConfig(config); err != nil {
		return "", fmt.Errorf("failed to pad app identity from config: %w", err)
	}

	name, err := identity.Format(namingSettings.TopicPattern, namingSettings.TopicDelimiter, map[string]string{
		"topicId": topicId,
	})
	if err != nil {
		return "", fmt.Errorf("kafka topic naming failed: %w", err)
	}

	return NormalizeKafkaName(name), nil
}

func BuildFullConsumerGroupId(config cfg.Config, groupId string) (string, error) {
	identity, err := cfg.GetAppIdentity(config)
	if err != nil {
		return "", fmt.Errorf("failed to get app identity from config: %w", err)
	}

	namingSettings := &KafkaNamingSettings{}
	if err := config.UnmarshalKey("kafka.naming", namingSettings); err != nil {
		return "", fmt.Errorf("failed to unmarshal kafka naming settings for key 'kafka.naming' to build kakfa consumer group id: %w", err)
	}

	name, err := identity.Format(namingSettings.GroupPattern, namingSettings.GroupDelimiter, map[string]string{
		"groupId": groupId,
	})
	if err != nil {
		return "", fmt.Errorf("kafka consumer group naming failed: %w", err)
	}

	return NormalizeKafkaName(name), nil
}
