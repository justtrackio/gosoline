package ddb

import (
	"fmt"
	"strings"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/cloud/aws"
	"github.com/justtrackio/gosoline/pkg/mdl"
)

type TableNamingSettings struct {
	Pattern string `cfg:"pattern,nodecode" default:"{app.tags.project}-{app.env}-{app.tags.family}-{app.tags.group}"`
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
//   - {app.env} - the Env field
//   - {app.name} - the App field
//   - {app.tags.<key>} - any tag from the Tags map
func GetTableNameWithSettings(tableSettings *Settings, namingSettings *TableNamingSettings) (string, error) {
	pattern := namingSettings.Pattern

	if tableSettings.TableNamingSettings.Pattern != "" {
		pattern = tableSettings.TableNamingSettings.Pattern
	}

	// Use internal formatting to allow flexible delimiters (e.g., dashes)
	tableName, err := formatTableName(tableSettings.ModelId, pattern)
	if err != nil {
		return "", fmt.Errorf("failed to format table name with pattern %q: %w", pattern, err)
	}

	return tableName, nil
}

func formatTableName(m mdl.ModelId, pattern string) (string, error) {
	result := pattern
	var missingTags []string

	// Simple extraction of placeholders like {foo}
	// We do a manual scan or reuse the logic.
	// Since we can't call internal mdl.extractPlaceholders, we'll reimplement a simple version or
	// iterate known placeholders. But dynamic tags {app.tags.*} make iteration hard.
	// Let's implement a simple placeholder extractor.

	placeholders := extractPlaceholders(pattern)
	for _, ph := range placeholders {
		var value string
		var ok bool

		switch {
		case ph == mdl.PlaceholderAppEnv:
			value = m.Env
			ok = m.Env != ""
			if !ok {
				return "", fmt.Errorf("pattern requires %s but it is empty", mdl.PlaceholderAppEnv)
			}
		case ph == mdl.PlaceholderAppName:
			value = m.App
			ok = m.App != ""
			if !ok {
				return "", fmt.Errorf("pattern requires %s but it is empty", mdl.PlaceholderAppName)
			}
		case strings.HasPrefix(ph, mdl.PlaceholderAppTags):
			tagKey := strings.TrimPrefix(ph, mdl.PlaceholderAppTags)
			if m.Tags != nil {
				value, ok = m.Tags[tagKey]
			}
			if !ok || value == "" {
				missingTags = append(missingTags, tagKey)

				continue
			}
		default:
			return "", fmt.Errorf("unknown placeholder {%s} in pattern %q", ph, pattern)
		}

		result = strings.ReplaceAll(result, "{"+ph+"}", value)
	}

	if len(missingTags) > 0 {
		return "", fmt.Errorf("missing required tags: %s", strings.Join(missingTags, ", "))
	}

	result = fmt.Sprintf("%s-%s", result, m.Name)

	return result, nil
}

func extractPlaceholders(pattern string) []string {
	var placeholders []string
	remaining := pattern

	for {
		start := strings.Index(remaining, "{")
		if start == -1 {
			break
		}

		end := strings.Index(remaining[start:], "}")
		if end == -1 {
			break
		}

		ph := remaining[start+1 : start+end]
		if ph != "" {
			placeholders = append(placeholders, ph)
		}

		remaining = remaining[start+end+1:]
	}

	return placeholders
}
