package kinesis

import (
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/cloud/aws"
)

type StreamNameSettingsAware interface {
	GetAppIdentity() cfg.AppIdentity
	GetClientName() string
	GetStreamName() string
}

type StreamNamingSettings struct {
	Pattern string `cfg:"pattern,nodecode" default:"{app.tags.project}-{app.env}-{app.tags.family}-{app.tags.group}-{streamName}"`
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

	identity := settings.GetAppIdentity()

	// Use NamingTemplate for strict placeholder validation and pattern-driven tag requirements
	tmpl := cfg.NewNamingTemplate(namingSettings.Pattern, "streamName")
	tmpl.WithResourceValue("streamName", settings.GetStreamName())

	name, err := tmpl.ValidateAndExpand(identity)
	if err != nil {
		return "", fmt.Errorf("kinesis stream naming failed: %w", err)
	}

	return Stream(name), nil
}
