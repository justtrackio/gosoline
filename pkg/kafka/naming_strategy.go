package kafka

import (
	"fmt"
	"strings"

	"github.com/justtrackio/gosoline/pkg/cfg"
)

type KafkaNamingSettings struct {
	TopicPattern string `cfg:"topic_pattern,nodecode" default:"{realm}-{topicId}"`
	GroupPattern string `cfg:"group_pattern,nodecode" default:"{realm}-{app}-{groupId}"`
}

func NormalizeKafkaName(name string) string {
	return strings.ReplaceAll(name, "_", "-")
}

// FQTopicName returns fully-qualified topic name.
func FQTopicName(config cfg.Config, appId cfg.AppId, topic string) (string, error) {
	namingSettings := &KafkaNamingSettings{}
	if err := config.UnmarshalKey("kafka.naming", namingSettings); err != nil {
		return "", fmt.Errorf("failed to unmarshal kafka naming settings for key 'kafka.naming' in FQTopicName: %w", err)
	}

	// Use AppId's ReplaceMacros method with topicId as extra macro
	extraMacros := []cfg.MacroValue{
		{"topicId", topic},
	}

	name := appId.ReplaceMacros(namingSettings.TopicPattern, extraMacros...)

	return NormalizeKafkaName(name), nil
}

func FQGroupId(config cfg.Config, appId cfg.AppId, groupId string) (string, error) {
	namingSettings := &KafkaNamingSettings{}
	if err := config.UnmarshalKey("kafka.naming", namingSettings); err != nil {
		return "", fmt.Errorf("failed to unmarshal kafka naming settings for key 'kafka.naming' in FQGroupId: %w", err)
	}

	// legacy naming support
	if groupId == "" {
		return appId.Application, nil
	}

	// Use AppId's ReplaceMacros method with groupId as extra macro
	extraMacros := []cfg.MacroValue{
		{"groupId", groupId},
	}

	name := appId.ReplaceMacros(namingSettings.GroupPattern, extraMacros...)

	return NormalizeKafkaName(name), nil
}
