package ddb

import (
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/cloud/aws"
)

type TableNamingSettings struct {
	TablePattern   string `cfg:"table_pattern,nodecode" default:"{app.namespace}-{name}"`
	TableDelimiter string `cfg:"table_delimiter" default:"-"`
}

func GetTableName(config cfg.Config, settings *Settings) (string, error) {
	namingSettings, err := GetTableNamingSettings(config, settings.ClientName)
	if err != nil {
		return "", fmt.Errorf("failed to get table naming settings for client %s: %w", settings.ClientName, err)
	}

	identity := cfg.Identity{
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
	if settings.TableNamingSettings.TableDelimiter != "" {
		namingSettings.TableDelimiter = settings.TableNamingSettings.TableDelimiter
	}

	return identity.Format(namingSettings.TablePattern, namingSettings.TableDelimiter, settings.ModelId.ToMap())
}

func GetTableNamingSettings(config cfg.Config, clientName string) (*TableNamingSettings, error) {
	if clientName == "" {
		clientName = "default"
	}

	namingKey := fmt.Sprintf("%s.naming", aws.GetClientConfigKey("dynamodb", clientName))
	defaultTablePatternKey := fmt.Sprintf("%s.naming.table_pattern", aws.GetClientConfigKey("dynamodb", "default"))
	defaultTableDelimiterKey := fmt.Sprintf("%s.naming.table_delimiter", aws.GetClientConfigKey("dynamodb", "default"))

	namingSettings := &TableNamingSettings{}
	err := config.UnmarshalKey(namingKey, namingSettings, []cfg.UnmarshalDefaults{
		cfg.UnmarshalWithDefaultsFromKey(defaultTablePatternKey, "table_pattern"),
		cfg.UnmarshalWithDefaultsFromKey(defaultTableDelimiterKey, "table_delimiter"),
	}...)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal table naming settings: %w", err)
	}

	return namingSettings, nil
}
