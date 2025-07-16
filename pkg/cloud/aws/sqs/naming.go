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
	Pattern string `cfg:"pattern,nodecode" default:"{realm}-{app}-{queueId}"`
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

	// Resolve realm pattern if it's used in the pattern
	realm := ""
	if strings.Contains(name, "{realm}") {
		var err error
		realm, err = appId.ResolveRealmPattern(config, "sqs", queueSettings.GetClientName())
		if err != nil {
			return "", fmt.Errorf("failed to resolve realm pattern for sqs: %w", err)
		}
	}
	
	// Use AppId's ReplaceMacros method with queueId and realm as extra macros
	extraMacros := []cfg.MacroValue{
		{"realm", realm},
		{"queueId", queueSettings.GetQueueId()},
	}

	name = appId.ReplaceMacros(name, extraMacros...)

	if queueSettings.IsFifoEnabled() {
		name += FifoSuffix
	}

	return name, nil
}
