package ddb

import (
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/cloud/aws"
)

type TableNamingSettings struct {
	TablePattern string `cfg:"table_pattern,nodecode" default:"{app.namespace}-{name}"`
	Delimiter    string `cfg:"delimiter" default:"-"`
}

func GetTableName(config cfg.Config, settings *Settings) (string, error) {
	namingSettings, err := GetTableNamingSettings(config, settings.ClientName)
	if err != nil {
		return "", fmt.Errorf("failed to get table naming settings for client %s: %w", settings.ClientName, err)
	}

	identity := cfg.AppIdentity{
		Env:  settings.ModelId.Env,
		Name: settings.ModelId.Name,
		Tags: settings.ModelId.Tags,
	}
	if err := identity.PadFromConfig(config); err != nil {
		return "", fmt.Errorf("failed to pad ModelId from config: %w", err)
	}

	if settings.TableNamingSettings.TablePattern != "" {
		namingSettings.TablePattern = settings.TableNamingSettings.TablePattern
	}

	return identity.Format(namingSettings.TablePattern, namingSettings.Delimiter, settings.ModelId.ToMap())
}

func GetTableNamingSettings(config cfg.Config, clientName string) (*TableNamingSettings, error) {
	if clientName == "" {
		clientName = "default"
	}

	namingKey := fmt.Sprintf("%s.naming", aws.GetClientConfigKey("dynamodb", clientName))
	defaultPatternKey := fmt.Sprintf("%s.naming.table_pattern", aws.GetClientConfigKey("dynamodb", "default"))
	namingSettings := &TableNamingSettings{}
	if err := config.UnmarshalKey(namingKey, namingSettings, cfg.UnmarshalWithDefaultsFromKey(defaultPatternKey, "table_pattern")); err != nil {
		return nil, fmt.Errorf("failed to unmarshal table naming settings: %w", err)
	}

	return namingSettings, nil
}
