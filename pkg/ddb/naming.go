package ddb

import (
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/cloud/aws"
)

type TableNamingSettings struct {
	Pattern string `cfg:"pattern,nodecode" default:"{realm}-{modelId}"`
}

func TableName(config cfg.Config, settings *Settings) (string, error) {
	namingSettings, err := GetTableNamingSettings(config, settings.ClientName)
	if err != nil {
		return "", fmt.Errorf("failed to get table naming settings for client %s: %w", settings.ClientName, err)
	}

	return GetTableNameWithSettings(settings, namingSettings), nil
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

func GetTableNameWithSettings(tableSettings *Settings, namingSettings *TableNamingSettings) string {
	tableName := namingSettings.Pattern

	if tableSettings.TableNamingSettings.Pattern != "" {
		tableName = tableSettings.TableNamingSettings.Pattern
	}

	// Convert ModelId to AppId for using the ReplaceMacros method
	appId := cfg.AppId{
		Project:     tableSettings.ModelId.Project,
		Environment: tableSettings.ModelId.Environment,
		Family:      tableSettings.ModelId.Family,
		Group:       tableSettings.ModelId.Group,
		Application: tableSettings.ModelId.Application,
		Realm:       tableSettings.ModelId.Realm,
	}

	// Use AppId's ReplaceMacros method with modelId as extra macro
	extraMacros := []cfg.MacroValue{
		{"modelId", tableSettings.ModelId.Name},
	}

	return appId.ReplaceMacros(tableName, extraMacros...)
}
