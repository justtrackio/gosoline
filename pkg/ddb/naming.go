package ddb

import (
	"fmt"
	"strings"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/cloud/aws"
)

type TableNamingSettings struct {
	Pattern string `cfg:"pattern,nodecode" default:"{project}-{env}-{family}-{group}-{modelId}"`
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

	values := map[string]string{
		"project": tableSettings.ModelId.Project,
		"env":     tableSettings.ModelId.Environment,
		"family":  tableSettings.ModelId.Family,
		"group":   tableSettings.ModelId.Group,
		"app":     tableSettings.ModelId.Application,
		"modelId": tableSettings.ModelId.Name,
	}

	for key, val := range values {
		templ := fmt.Sprintf("{%s}", key)
		tableName = strings.ReplaceAll(tableName, templ, val)
	}

	return tableName
}
