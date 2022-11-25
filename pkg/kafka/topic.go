package kafka

import (
	"fmt"
	"strings"

	"github.com/justtrackio/gosoline/pkg/cfg"
)

func NormalizeTopicName(name string) string {
	return strings.ReplaceAll(name, "_", "-")
}

// FQTopicName returns fully-qualified topic name.
func FQTopicName(appId cfg.AppId, topic string) string {
	return NormalizeTopicName(namingStrategy(appId, topic))
}

// DLTopicName returns dead letter topic name.
func DLTopicName(topic, groupID string) string {
	return NormalizeTopicName(
		fmt.Sprintf(
			"%s.%s.dl",
			topic,
			groupID,
		))
}

type NamingStrategy func(appId cfg.AppId, topic string) string

func WithNamingStrategy(strategy NamingStrategy) {
	namingStrategy = strategy
}

var namingStrategy = func(appId cfg.AppId, topic string) string {
	return fmt.Sprintf("%s-%s", appId.Environment, topic)
}
