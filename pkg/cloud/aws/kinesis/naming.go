package kinesis

import (
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/cloud/aws"
)

type StreamNameSettingsAware interface {
	GetIdentity() cfg.Identity
	GetClientName() string
	GetStreamName() string
}

type KinsumerNamingSettings struct {
	StreamPattern              string `cfg:"stream_pattern,nodecode" default:"{app.namespace}-{streamName}"`
	StreamDelimiter            string `cfg:"stream_delimiter" default:"-"`
	MetadataTablePattern       string `cfg:"metadata_table_pattern,nodecode" default:"{app.namespace}-kinsumer-metadata"`
	MetadataTableDelimiter     string `cfg:"metadata_table_delimiter" default:"-"`
	MetadataNamespacePattern   string `cfg:"metadata_namespace_pattern,nodecode" default:"{app.namespace}-{app.name}"`
	MetadataNamespaceDelimiter string `cfg:"metadata_namespace_delimiter" default:"-"`
}

func GetStreamName(config cfg.Config, settings StreamNameSettingsAware) (Stream, error) {
	var err error
	var namingSettings *KinsumerNamingSettings

	if namingSettings, err = readNamingSettings(config, settings); err != nil {
		return "", fmt.Errorf("failed to read naming settings: %w", err)
	}

	identity := settings.GetIdentity()
	if err := identity.PadFromConfig(config); err != nil {
		return "", fmt.Errorf("failed to pad app identity from config: %w", err)
	}

	name, err := identity.Format(namingSettings.StreamPattern, namingSettings.StreamDelimiter, map[string]string{
		"streamName": settings.GetStreamName(),
	})
	if err != nil {
		return "", fmt.Errorf("failed to format kinesis naming settings for %s: %w", settings.GetStreamName(), err)
	}

	return Stream(name), nil
}

func GetMetadataTableName(config cfg.Config, settings StreamNameSettingsAware) (string, error) {
	var err error
	var namingSettings *KinsumerNamingSettings

	if namingSettings, err = readNamingSettings(config, settings); err != nil {
		return "", fmt.Errorf("failed to read naming settings: %w", err)
	}

	identity := settings.GetIdentity()
	if err := identity.PadFromConfig(config); err != nil {
		return "", fmt.Errorf("failed to pad app identity from config: %w", err)
	}

	name, err := identity.Format(namingSettings.MetadataTablePattern, namingSettings.MetadataTableDelimiter)
	if err != nil {
		return "", fmt.Errorf("failed to format kinesis metadata table naming settings: %w", err)
	}

	return name, nil
}

func readNamingSettings(config cfg.Config, settings StreamNameSettingsAware) (*KinsumerNamingSettings, error) {
	if settings.GetClientName() == "" {
		return nil, fmt.Errorf("the client name shouldn't be empty")
	}

	namingKey := fmt.Sprintf("%s.naming", aws.GetClientConfigKey("kinesis", settings.GetClientName()))
	defaultNamingKey := fmt.Sprintf("%s.naming", aws.GetClientConfigKey("kinesis", "default"))

	defaults := []cfg.UnmarshalDefaults{
		cfg.UnmarshalWithDefaultsFromKey(fmt.Sprintf("%s.stream_pattern", defaultNamingKey), "stream_pattern"),
		cfg.UnmarshalWithDefaultsFromKey(fmt.Sprintf("%s.stream_delimiter", defaultNamingKey), "stream_delimiter"),
		cfg.UnmarshalWithDefaultsFromKey(fmt.Sprintf("%s.metadata_table_pattern", defaultNamingKey), "metadata_table_pattern"),
		cfg.UnmarshalWithDefaultsFromKey(fmt.Sprintf("%s.metadata_table_delimiter", defaultNamingKey), "metadata_table_delimiter"),
		cfg.UnmarshalWithDefaultsFromKey(fmt.Sprintf("%s.metadata_namespace_pattern", defaultNamingKey), "metadata_namespace_pattern"),
		cfg.UnmarshalWithDefaultsFromKey(fmt.Sprintf("%s.metadata_namespace_delimiter", defaultNamingKey), "metadata_namespace_delimiter"),
	}

	namingSettings := &KinsumerNamingSettings{}
	if err := config.UnmarshalKey(namingKey, namingSettings, defaults...); err != nil {
		return nil, fmt.Errorf("failed to unmarshal kinesis naming settings for %s: %w", namingKey, err)
	}

	return namingSettings, nil
}
