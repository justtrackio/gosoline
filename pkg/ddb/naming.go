package ddb

import (
	"crypto/sha256"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/cloud/aws"
)

const maxTableNameLength = 255

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

	name, err := identity.Format(namingSettings.TablePattern, namingSettings.TableDelimiter, settings.ModelId.ToMap())
	if err != nil {
		return "", err
	}
	if len(name) > maxTableNameLength {
		// Hash the full resolved name so truncation preserves model and environment isolation.
		digest := sha256.Sum256([]byte(name))
		name = fmt.Sprintf("%s-%x", name[:maxTableNameLength-1-2*sha256.Size], digest)
	}

	return name, nil
}

func GetTableNamingSettings(config cfg.Config, clientName string) (*TableNamingSettings, error) {
	var namingSettings TableNamingSettings
	var err error

	if clientName == "" {
		clientName = "default"
	}

	namingKey := fmt.Sprintf("%s.naming", aws.GetClientConfigKey("dynamodb", clientName))
	defaultTablePatternKey := fmt.Sprintf("%s.naming.table_pattern", aws.GetClientConfigKey("dynamodb", "default"))
	defaultTableDelimiterKey := fmt.Sprintf("%s.naming.table_delimiter", aws.GetClientConfigKey("dynamodb", "default"))

	if namingSettings, err = cfg.GetSettings[TableNamingSettings](config, namingKey, []cfg.UnmarshalDefaults{
		cfg.UnmarshalWithDefaultsFromKey(defaultTablePatternKey, "table_pattern"),
		cfg.UnmarshalWithDefaultsFromKey(defaultTableDelimiterKey, "table_delimiter"),
	}...); err != nil {
		return nil, fmt.Errorf("failed to unmarshal table naming settings: %w", err)
	}

	return &namingSettings, nil
}
