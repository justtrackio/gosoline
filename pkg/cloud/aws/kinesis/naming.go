package kinesis

import (
	"fmt"
	"strings"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/cloud/aws"
)

type StreamNameSettingsAware interface {
	GetAppId() cfg.AppId
	GetClientName() string
	GetStreamName() string
}

type StreamNamingSettings struct {
	Pattern string `cfg:"pattern,nodecode" default:"{realm}-{streamName}"`
}

func GetStreamName(config cfg.Config, settings StreamNameSettingsAware) (Stream, error) {
	if settings.GetClientName() == "" {
		return "", fmt.Errorf("the client name shouldn't be empty")
	}

	namingKey := fmt.Sprintf("%s.naming", aws.GetClientConfigKey("kinesis", settings.GetClientName()))
	defaultPatternKey := fmt.Sprintf("%s.naming.pattern", aws.GetClientConfigKey("kinesis", "default"))
	namingSettings := &StreamNamingSettings{}
	if err := config.UnmarshalKey(namingKey, namingSettings, cfg.UnmarshalWithDefaultsFromKey(defaultPatternKey, "pattern")); err != nil {
		return "", fmt.Errorf("failed to unmarshal kinesis naming settings for %s: %w", namingKey, err)
	}

	appId := settings.GetAppId()
	name := namingSettings.Pattern

	// Resolve realm pattern if it's used in the pattern
	realm := ""
	if strings.Contains(name, "{realm}") {
		var err error
		realm, err = appId.ResolveRealmPattern(config, "kinesis", settings.GetClientName())
		if err != nil {
			return "", fmt.Errorf("failed to resolve realm pattern for kinesis: %w", err)
		}
	}

	// Use AppId's ReplaceMacros method with streamName and realm as extra macros
	extraMacros := []cfg.MacroValue{
		{"realm", realm},
		{"streamName", settings.GetStreamName()},
	}

	return Stream(appId.ReplaceMacros(name, extraMacros...)), nil
}
