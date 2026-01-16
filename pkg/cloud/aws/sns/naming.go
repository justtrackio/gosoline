package sns

import (
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/cloud/aws"
)

type TopicNameSettingsAware interface {
	GetAppIdentity() cfg.AppIdentity
	GetClientName() string
	GetTopicId() string
}

type TopicNameSettings struct {
	AppIdentity cfg.AppIdentity
	ClientName  string
	TopicId     string
}

func (s TopicNameSettings) GetAppIdentity() cfg.AppIdentity {
	return s.AppIdentity
}

func (s TopicNameSettings) GetClientName() string {
	return s.ClientName
}

func (s TopicNameSettings) GetTopicId() string {
	return s.TopicId
}

type TopicNamingSettings struct {
	Pattern string `cfg:"pattern,nodecode" default:"{app.tags.project}-{app.env}-{app.tags.family}-{app.tags.group}-{topicId}"`
}

func GetTopicName(config cfg.Config, topicSettings TopicNameSettingsAware) (string, error) {
	if topicSettings.GetClientName() == "" {
		return "", fmt.Errorf("the client name shouldn't be empty")
	}

	namingKey := fmt.Sprintf("%s.naming", aws.GetClientConfigKey("sns", topicSettings.GetClientName()))
	defaultPatternKey := fmt.Sprintf("%s.naming.pattern", aws.GetClientConfigKey("sns", "default"))

	namingSettings := &TopicNamingSettings{}
	if err := config.UnmarshalKey(namingKey, namingSettings, cfg.UnmarshalWithDefaultsFromKey(defaultPatternKey, "pattern")); err != nil {
		return "", fmt.Errorf("failed to unmarshal sns naming settings for %s: %w", namingKey, err)
	}

	return config.FormatString(namingSettings.Pattern, topicSettings.GetAppIdentity().ToMap(), map[string]string{
		"topicId": topicSettings.GetTopicId(),
	})
}
