package sns

import (
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/cloud/aws"
)

type TopicNameSettingsAware interface {
	GetIdentity() cfg.Identity
	GetClientName() string
	GetTopicId() string
}

type TopicNameSettings struct {
	Identity   cfg.Identity
	ClientName string
	TopicId    string
}

func (s TopicNameSettings) GetIdentity() cfg.Identity {
	return s.Identity
}

func (s TopicNameSettings) GetClientName() string {
	return s.ClientName
}

func (s TopicNameSettings) GetTopicId() string {
	return s.TopicId
}

type TopicNamingSettings struct {
	TopicPattern   string `cfg:"topic_pattern,nodecode" default:"{app.namespace}-{topicId}"`
	TopicDelimiter string `cfg:"topic_delimiter" default:"-"`
}

func GetTopicName(config cfg.Config, topicSettings TopicNameSettingsAware) (string, error) {
	if topicSettings.GetClientName() == "" {
		return "", fmt.Errorf("the client name shouldn't be empty")
	}

	namingKey := fmt.Sprintf("%s.naming", aws.GetClientConfigKey("sns", topicSettings.GetClientName()))
	defaultPatternKey := fmt.Sprintf("%s.naming.topic_pattern", aws.GetClientConfigKey("sns", "default"))

	namingSettings := &TopicNamingSettings{}
	if err := config.UnmarshalKey(namingKey, namingSettings, cfg.UnmarshalWithDefaultsFromKey(defaultPatternKey, "topic_pattern")); err != nil {
		return "", fmt.Errorf("failed to unmarshal sns naming settings for %s: %w", namingKey, err)
	}

	identity := topicSettings.GetIdentity()
	if err := identity.PadFromConfig(config); err != nil {
		return "", fmt.Errorf("failed to pad app identity from config: %w", err)
	}

	name, err := identity.Format(namingSettings.TopicPattern, namingSettings.TopicDelimiter, map[string]string{
		"topicId": topicSettings.GetTopicId(),
	})
	if err != nil {
		return "", fmt.Errorf("sns topic naming failed: %w", err)
	}

	return name, nil
}
