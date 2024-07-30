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
	Pattern string `cfg:"pattern,nodecode" default:"{project}-{env}-{family}-{group}-{topicId}"`
}

func GetTopicName(config cfg.Config, topicSettings TopicNameSettingsAware) (string, error) {
	if len(topicSettings.GetClientName()) == 0 {
		return "", fmt.Errorf("the client name shouldn't be empty")
	}

	namingKey := fmt.Sprintf("%s.naming", aws.GetClientConfigKey("sns", topicSettings.GetClientName()))
	namingSettings := &TopicNamingSettings{}
	config.UnmarshalKey(namingKey, namingSettings)

	name := namingSettings.Pattern
	appId := topicSettings.GetAppId()
	values := map[string]string{
		"project": appId.Project,
		"env":     appId.Environment,
		"family":  appId.Family,
		"group":   appId.Group,
		"app":     appId.Application,
		"topicId": topicSettings.GetTopicId(),
	}

	for key, val := range values {
		templ := fmt.Sprintf("{%s}", key)
		name = strings.ReplaceAll(name, templ, val)
	}

	return name, nil
}
