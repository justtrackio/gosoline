package kafka

import (
	"fmt"
	"strings"

	"github.com/justtrackio/gosoline/pkg/cfg"
)

type KafkaNamingSettings struct {
	TopicPattern string `cfg:"topic_pattern,nodecode" default:"{env}-{topicId}"`
	GroupPattern string `cfg:"group_pattern,nodecode" default:"{env}-{app}-{groupId}"`
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

	name := namingSettings.TopicPattern
	values := map[string]string{
		"project": appId.Project,
		"env":     appId.Environment,
		"family":  appId.Family,
		"group":   appId.Group,
		"app":     appId.Application,
		"topicId": topic,
	}

	for key, val := range values {
		name = strings.ReplaceAll(name, fmt.Sprintf("{%s}", key), val)
	}

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

	name := namingSettings.GroupPattern
	values := map[string]string{
		"project": appId.Project,
		"env":     appId.Environment,
		"family":  appId.Family,
		"group":   appId.Group,
		"app":     appId.Application,
		"groupId": groupId,
	}

	for key, val := range values {
		name = strings.ReplaceAll(name, fmt.Sprintf("{%s}", key), val)
	}

	return NormalizeKafkaName(name), nil
}
