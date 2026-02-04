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
	StreamPattern     string `cfg:"stream_pattern,nodecode" default:"{app.tags.project}-{app.env}-{app.tags.family}-{app.tags.group}-{streamName}"`
	StreamDelimiter   string `cfg:"stream_delimiter" default:"-"`
	MetadataPattern   string `cfg:"metadata_pattern,nodecode" default:"{app.env}-kinsumer-metadata"`
	MetadataDelimiter string `cfg:"metadata_delimiter" default:"-"`
}

func GetStreamName(config cfg.Config, settings StreamNameSettingsAware) (Stream, error) {
	var err error
	var namingSettings *StreamNamingSettings

	if namingSettings, err = readNamingSettings(config, settings); err != nil {
		return "", fmt.Errorf("failed to read naming settings: %w", err)
	}

	name, err := settings.GetAppIdentity().Format(namingSettings.StreamPattern, namingSettings.StreamDelimiter, map[string]string{
		"streamName": settings.GetStreamName(),
	})
	if err != nil {
		return "", fmt.Errorf("failed to format kinesis naming settings for %s: %w", settings.GetStreamName(), err)
	}

	return Stream(name), nil
}

func GetMetadataTableName(config cfg.Config, settings StreamNameSettingsAware) (string, error) {
	var err error
	var namingSettings *StreamNamingSettings

	if namingSettings, err = readNamingSettings(config, settings); err != nil {
		return "", fmt.Errorf("failed to read naming settings: %w", err)
	}

	name, err := settings.GetAppIdentity().Format(namingSettings.MetadataPattern, namingSettings.MetadataDelimiter)
	if err != nil {
		return "", fmt.Errorf("failed to format kinesis metadata table naming settings: %w", err)
	}

	return name, nil
}

func readNamingSettings(config cfg.Config, settings StreamNameSettingsAware) (*StreamNamingSettings, error) {
	if settings.GetClientName() == "" {
		return nil, fmt.Errorf("the client name shouldn't be empty")
	}

	namingKey := fmt.Sprintf("%s.naming", aws.GetClientConfigKey("kinesis", settings.GetClientName()))
	defaultNamingKey := fmt.Sprintf("%s.naming", aws.GetClientConfigKey("kinesis", "default"))
	defaultStreamPatternKey := fmt.Sprintf("%s.stream_pattern", defaultNamingKey)
	defaultMetadataPatternKey := fmt.Sprintf("%s.metadata_pattern", defaultNamingKey)

	namingSettings := &StreamNamingSettings{}
	if err := config.UnmarshalKey(namingKey, namingSettings, cfg.UnmarshalWithDefaultsFromKey(defaultStreamPatternKey, "stream_pattern"), cfg.UnmarshalWithDefaultsFromKey(defaultMetadataPatternKey, "metadata_pattern")); err != nil {
		return nil, fmt.Errorf("failed to unmarshal kinesis naming settings for %s: %w", namingKey, err)
	}

	return namingSettings, nil
}
