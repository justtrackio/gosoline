package kafka

import (
	"fmt"
	"strings"

	"github.com/justtrackio/gosoline/pkg/cfg"
)

type KafkaNamingSettings struct {
	TopicPattern string `cfg:"topic_pattern,nodecode" default:"{project}-{env}-{family}-{group}-{topicId}"`
	GroupPattern string `cfg:"group_pattern,nodecode" default:"{project}-{env}-{family}-{group}-{groupId}"`
}

func NormalizeKafkaName(name string) string {
	return strings.ReplaceAll(name, "_", "-")
}

func BuildFullTopicName(config cfg.Config, appId cfg.AppId, topicId string) (string, error) {
	namingSettings := &KafkaNamingSettings{}
	if err := config.UnmarshalKey("kafka.naming", namingSettings); err != nil {
		return "", fmt.Errorf("failed to unmarshal kafka naming settings for key 'kafka.naming' to build kafka topic name: %w", err)
	}

	name := namingSettings.TopicPattern
	values := map[string]string{
		"project": appId.Project,
		"env":     appId.Environment,
		"family":  appId.Family,
		"group":   appId.Group,
		"app":     appId.Application,
		"topicId": topicId,
	}

	for key, val := range values {
		name = strings.ReplaceAll(name, fmt.Sprintf("{%s}", key), val)
	}

	return NormalizeKafkaName(name), nil
}

func BuildFullConsumerGroupId(config cfg.Config, appId cfg.AppId, groupId string) (string, error) {
	namingSettings := &KafkaNamingSettings{}
	if err := config.UnmarshalKey("kafka.naming", namingSettings); err != nil {
		return "", fmt.Errorf("failed to unmarshal kafka naming settings for key 'kafka.naming' to build kakfa consumer group id: %w", err)
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
