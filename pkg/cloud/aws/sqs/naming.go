package sqs

import (
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/cloud/aws"
)

type QueueNameSettingsAware interface {
	GetIdentity() cfg.Identity
	GetClientName() string
	GetQueueId() string
	IsFifoEnabled() bool
}

type QueueNameSettings struct {
	Identity    cfg.Identity
	ClientName  string
	FifoEnabled bool
	QueueId     string
}

func (s QueueNameSettings) GetIdentity() cfg.Identity {
	return s.Identity
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
	QueuePattern   string `cfg:"queue_pattern,nodecode" default:"{app.namespace}-{queueId}"`
	QueueDelimiter string `cfg:"queue_delimiter" default:"-"`
}

func GetQueueName(config cfg.Config, queueSettings QueueNameSettingsAware) (string, error) {
	if queueSettings.GetClientName() == "" {
		return "", fmt.Errorf("the client name shouldn't be empty")
	}

	namingKey := fmt.Sprintf("%s.naming", aws.GetClientConfigKey("sqs", queueSettings.GetClientName()))
	defaultPatternKey := fmt.Sprintf("%s.naming.queue_pattern", aws.GetClientConfigKey("sqs", "default"))
	namingSettings := &QueueNamingSettings{}
	if err := config.UnmarshalKey(namingKey, namingSettings, cfg.UnmarshalWithDefaultsFromKey(defaultPatternKey, "queue_pattern")); err != nil {
		return "", fmt.Errorf("failed to unmarshal sqs naming settings for %s: %w", namingKey, err)
	}

	identity := queueSettings.GetIdentity()
	if err := identity.PadFromConfig(config); err != nil {
		return "", fmt.Errorf("failed to pad app identity from config: %w", err)
	}

	name, err := identity.Format(namingSettings.QueuePattern, namingSettings.QueueDelimiter, map[string]string{
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
