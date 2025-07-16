package sns

import (
	"fmt"
	"strings"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/cloud/aws"
)

type TopicNameSettingsAware interface {
	GetAppId() cfg.AppId
	GetClientName() string
	GetTopicId() string
}

type TopicNameSettings struct {
	AppId      cfg.AppId
	ClientName string
	TopicId    string
}

func (s TopicNameSettings) GetAppId() cfg.AppId {
	return s.AppId
}

func (s TopicNameSettings) GetClientName() string {
	return s.ClientName
}

func (s TopicNameSettings) GetTopicId() string {
	return s.TopicId
}

type TopicNamingSettings struct {
	Pattern string `cfg:"pattern,nodecode" default:"{realm}-{topicId}"`
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

	name := namingSettings.Pattern
	appId := topicSettings.GetAppId()

	// Resolve realm pattern if it's used in the pattern
	realm := ""
	if strings.Contains(name, "{realm}") {
		var err error
		realm, err = appId.ResolveRealmPattern(config, "sns", topicSettings.GetClientName())
		if err != nil {
			return "", fmt.Errorf("failed to resolve realm pattern for sns: %w", err)
		}
	}
	
	// Use AppId's ReplaceMacros method with topicId and realm as extra macros
	extraMacros := []cfg.MacroValue{
		{"realm", realm},
		{"topicId", topicSettings.GetTopicId()},
	}

	return appId.ReplaceMacros(name, extraMacros...), nil
}
