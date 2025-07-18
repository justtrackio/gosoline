package sqs

import (
	"fmt"
	"strings"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/cloud/aws"
)

type QueueNameSettingsAware interface {
	GetAppId() cfg.AppId
	GetClientName() string
	GetQueueId() string
	IsFifoEnabled() bool
}

type QueueNameSettings struct {
	AppId       cfg.AppId
	ClientName  string
	FifoEnabled bool
	QueueId     string
}

func (s QueueNameSettings) GetAppId() cfg.AppId {
	return s.AppId
}

func (s QueueNameSettings) GetClientName() string {
	return s.ClientName
}

func (s QueueNameSettings) IsFifoEnabled() bool {
	return s.FifoEnabled
}

func (s QueueNameSettings) GetQueueId() string {
	return s.QueueId
}

type QueueNamingSettings struct {
	Pattern string `cfg:"pattern,nodecode" default:"{project}-{env}-{family}-{group}-{queueId}"`
}

func GetQueueName(config cfg.Config, queueSettings QueueNameSettingsAware) (string, error) {
	if queueSettings.GetClientName() == "" {
		return "", fmt.Errorf("the client name shouldn't be empty")
	}

	namingKey := fmt.Sprintf("%s.naming", aws.GetClientConfigKey("sqs", queueSettings.GetClientName()))
	defaultPatternKey := fmt.Sprintf("%s.naming.pattern", aws.GetClientConfigKey("sqs", "default"))
	namingSettings := &QueueNamingSettings{}
	if err := config.UnmarshalKey(namingKey, namingSettings, cfg.UnmarshalWithDefaultsFromKey(defaultPatternKey, "pattern")); err != nil {
		return "", fmt.Errorf("failed to unmarshal sqs naming settings for %s: %w", namingKey, err)
	}

	name := namingSettings.Pattern
	appId := queueSettings.GetAppId()
	values := map[string]string{
		"project": appId.Project,
		"env":     appId.Environment,
		"family":  appId.Family,
		"group":   appId.Group,
		"app":     appId.Application,
		"queueId": queueSettings.GetQueueId(),
	}

	for key, val := range values {
		templ := fmt.Sprintf("{%s}", key)
		name = strings.ReplaceAll(name, templ, val)
	}

	if queueSettings.IsFifoEnabled() {
		name += FifoSuffix
	}

	return name, nil
}
