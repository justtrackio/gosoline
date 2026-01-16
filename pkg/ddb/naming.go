package ddb

import (
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/cloud/aws"
	"github.com/justtrackio/gosoline/pkg/mdl"
)

type TableNamingSettings struct {
	Pattern string `cfg:"pattern,nodecode" default:"{app.tags.project}-{app.env}-{app.tags.family}-{app.tags.group}-{modelId}"`
}

func TableName(config cfg.Config, settings *Settings) (string, error) {
	namingSettings, err := GetTableNamingSettings(config, settings.ClientName)
	if err != nil {
		return "", fmt.Errorf("failed to get table naming settings for client %s: %w", settings.ClientName, err)
	}

	// Pad the ModelId from config to fill in missing values like Env, App, and Tags
	if err := settings.ModelId.PadFromConfig(config); err != nil {
		return "", fmt.Errorf("failed to pad ModelId from config: %w", err)
	}

	return GetTableNameWithSettings(settings, namingSettings)
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

// GetTableNameWithSettings formats the table name using the ModelId and naming pattern.
// The pattern supports the same placeholders as ModelId.Format:
//   - {modelId} - the model's Name
//   - {app.env} - the Env field
//   - {app.name} - the App field
//   - {app.tags.<key>} - any tag from the Tags map
func GetTableNameWithSettings(tableSettings *Settings, namingSettings *TableNamingSettings) (string, error) {
	pattern := namingSettings.Pattern

	if tableSettings.TableNamingSettings.Pattern != "" {
		pattern = tableSettings.TableNamingSettings.Pattern
	}

	// Use FormatModelIdWithPattern to expand the pattern
	tableName, err := mdl.FormatModelIdWithPattern(tableSettings.ModelId, pattern)
	if err != nil {
		return "", fmt.Errorf("failed to format table name with pattern %q: %w", pattern, err)
	}

	return tableName, nil
}
