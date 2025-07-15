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
	
	// Resolve realm if it's used in the pattern
	realm := ""
	if strings.Contains(name, "{realm}") {
		var err error
		realm, err = aws.ResolveRealm(config, appId, "sns", topicSettings.GetClientName())
		if err != nil {
			return "", fmt.Errorf("failed to resolve realm for sns: %w", err)
		}
	}
	
	values := map[string]string{
		"project": appId.Project,
		"env":     appId.Environment,
		"family":  appId.Family,
		"group":   appId.Group,
		"app":     appId.Application,
		"topicId": topicSettings.GetTopicId(),
		"realm":   realm,
	}

	for key, val := range values {
		templ := fmt.Sprintf("{%s}", key)
		name = strings.ReplaceAll(name, templ, val)
	}

	return name, nil
}
