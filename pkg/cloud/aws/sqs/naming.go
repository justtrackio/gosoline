package sqs

import (
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/cloud/aws"
)

type QueueNameSettingsAware interface {
	GetAppIdentity() cfg.AppIdentity
	GetClientName() string
	GetQueueId() string
	IsFifoEnabled() bool
}

type QueueNameSettings struct {
	AppIdentity cfg.AppIdentity
	ClientName  string
	FifoEnabled bool
	QueueId     string
}

func (s QueueNameSettings) GetAppIdentity() cfg.AppIdentity {
	return s.AppIdentity
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
	Pattern   string `cfg:"pattern,nodecode" default:"{app.tags.project}-{app.env}-{app.tags.family}-{app.tags.group}-{queueId}"`
	Delimiter string `cfg:"delimiter" default:"-"`
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

	name, err := queueSettings.GetAppIdentity().Format(namingSettings.Pattern, namingSettings.Delimiter, map[string]string{
		"queueId": queueSettings.GetQueueId(),
	})
	if err != nil {
		return "", fmt.Errorf("sqs queue naming failed: %w", err)
	}

	if queueSettings.IsFifoEnabled() {
		name += FifoSuffix
	}

	return name, nil
}
