package ddb

import (
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/cloud/aws"
)

type TableNamingSettings struct {
	Pattern   string `cfg:"pattern,nodecode" default:"{app.namespace}-{name}"`
	Delimiter string `cfg:"delimiter" default:"-"`
}

func GetTableName(config cfg.Config, settings *Settings) (string, error) {
	namingSettings, err := GetTableNamingSettings(config, settings.ClientName)
	if err != nil {
		return "", fmt.Errorf("failed to get table naming settings for client %s: %w", settings.ClientName, err)
	}

	identity := cfg.AppIdentity{
		Env:       settings.ModelId.Env,
		Name:      settings.ModelId.Name,
		Tags:      settings.ModelId.Tags,
	}
	if err := identity.PadFromConfig(config); err != nil {
		return "", fmt.Errorf("failed to pad ModelId from config: %w", err)
	}

	if settings.TableNamingSettings.Pattern != "" {
		namingSettings.Pattern = settings.TableNamingSettings.Pattern
	}

	return identity.Format(namingSettings.Pattern, namingSettings.Delimiter, settings.ModelId.ToMap())
}

func GetTableNamingSettings(config cfg.Config, clientName string) (*TableNamingSettings, error) {
	if clientName == "" {
		clientName = "default"
	}

	namingKey := fmt.Sprintf("%s.naming", aws.GetClientConfigKey("dynamodb", clientName))
	defaultPatternKey := fmt.Sprintf("%s.naming.pattern", aws.GetClientConfigKey("dynamodb", "default"))
	namingSettings := &TableNamingSettings{}
	if err := config.UnmarshalKey(namingKey, namingSettings, cfg.UnmarshalWithDefaultsFromKey(defaultPatternKey, "pattern")); err != nil {
		return nil, fmt.Errorf("failed to unmarshal table naming settings: %w", err)
	}

	return namingSettings, nil
}
