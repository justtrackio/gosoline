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
	
	// Resolve realm if it's used in the pattern
	realm := ""
	if strings.Contains(name, "{realm}") {
		var err error
		realm, err = cfg.ResolveRealm(config, appId, "sqs", queueSettings.GetClientName())
		if err != nil {
			return "", fmt.Errorf("failed to resolve realm for sqs: %w", err)
		}
	}
	
	// Use slice of MacroValue with realm first for proper resolution order
	values := []cfg.MacroValue{
		{"realm", realm},
		{"project", appId.Project},
		{"env", appId.Environment},
		{"family", appId.Family},
		{"group", appId.Group},
		{"app", appId.Application},
		{"queueId", queueSettings.GetQueueId()},
	}

	name = cfg.ReplaceMacros(name, values)

	if queueSettings.IsFifoEnabled() {
		name += FifoSuffix
	}

	return name, nil
}
