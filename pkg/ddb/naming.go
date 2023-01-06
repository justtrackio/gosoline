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

func TableName(config cfg.Config, settings *Settings) string {
	namingSettings := GetTableNamingSettings(config, settings.ClientName)

	return GetTableNameWithSettings(settings, namingSettings)
}

func GetTableNamingSettings(config cfg.Config, clientName string) *TableNamingSettings {
	if len(clientName) == 0 {
		clientName = "default"
	}

	namingKey := fmt.Sprintf("%s.naming", aws.GetClientConfigKey("dynamodb", clientName))
	namingSettings := &TableNamingSettings{}
	config.UnmarshalKey(namingKey, namingSettings)

	return namingSettings
}

func GetTableNameWithSettings(tableSettings *Settings, namingSettings *TableNamingSettings) string {
	tableName := namingSettings.Pattern

	if len(tableSettings.TableNamingSettings.Pattern) > 0 {
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
