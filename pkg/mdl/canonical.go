package mdl

import (
	"fmt"
)

// RequiredModelIdPattern reads the model id pattern from config.
// The pattern must be set in config at "app.model_id.pattern".
// Returns an error if:
//   - the config key is missing or empty
//   - the pattern is invalid (unknown placeholders, unparseable)
func RequiredModelIdPattern(config ConfigProvider) (string, error) {
	pattern, err := config.GetString(ConfigKeyModelIdPattern)
	if err != nil {
		return "", fmt.Errorf("%s must be set: %w", ConfigKeyModelIdPattern, err)
	}

	if pattern == "" {
		return "", fmt.Errorf("%s must not be empty", ConfigKeyModelIdPattern)
	}

	if err := validateModelIdPattern(pattern); err != nil {
		return "", fmt.Errorf("invalid %s: %w", ConfigKeyModelIdPattern, err)
	}

	return pattern, nil
}

// ParseCanonicalModelId parses a string into a ModelId using the required pattern from config.
// This is the only supported way to parse a canonical model id string.
//
// Returns an error if:
//   - the pattern is not configured or invalid
//   - the string doesn't match the expected pattern structure
func ParseCanonicalModelId(config ConfigProvider, s string) (ModelId, error) {
	pattern, err := RequiredModelIdPattern(config)
	if err != nil {
		return ModelId{}, err
	}

	id, err := modelIdFromStringWithPattern(pattern, s)
	if err != nil {
		return ModelId{}, fmt.Errorf("failed to parse model id %q with pattern %q: %w", s, pattern, err)
	}

	return id, nil
}

// DebugModelIdString returns a debug representation of a ModelId.
// This is safe to use for logging and does not require config.
// It should NOT be used for routing, persistence keys, or message attributes.
func DebugModelIdString(id ModelId) string {
	return fmt.Sprintf("ModelId{Name:%q, Env:%q, App:%q, Tags:%v}", id.Name, id.Env, id.App, id.Tags)
}

// FormatModelIdWithPattern formats a ModelId using the given pattern.
// Use this when a service (e.g., DynamoDB table naming) has its own pattern configuration.
//
// The pattern supports the same placeholders:
//   - {modelId} - the model's Name
//   - {app.env} - the Env field
//   - {app.name} - the App field
//   - {app.tags.<key>} - any tag from the Tags map
//
// Returns an error if:
//   - the pattern is invalid (unknown placeholders)
//   - the pattern references fields/tags that are missing from the ModelId
func FormatModelIdWithPattern(id ModelId, pattern string) (string, error) {
	result, err := id.format(pattern)
	if err != nil {
		return "", fmt.Errorf("failed to format model id with pattern %q: %w", pattern, err)
	}

	return result, nil
}

// LegacyModelIdPattern is the old default format for ModelId strings.
// This pattern is used for backward compatibility in tests and internal map keys.
const LegacyModelIdPattern = "{app.tags.project}.{app.tags.family}.{app.tags.group}.{modelId}"

// ParseLegacyModelId parses a string using the legacy format "project.family.group.name".
// This is intended for test utilities and backward compatibility scenarios where
// config-driven patterns are not available.
//
// For production code, prefer ParseCanonicalModelId with proper config.
func ParseLegacyModelId(s string) (ModelId, error) {
	return modelIdFromStringWithPattern(LegacyModelIdPattern, s)
}

// FormatLegacyModelIdString formats a ModelId using the legacy format.
// This is intended for backward compatibility scenarios and internal map keys.
// Missing tags are replaced with empty strings.
//
// For production code, call PadFromConfig once, then use Format().
func FormatLegacyModelIdString(id ModelId) string {
	// Initialize tags if nil to avoid nil map access
	if id.Tags == nil {
		id.Tags = make(map[string]string)
	}

	// Use direct string formatting to allow empty tags (format() would error on missing tags)
	return fmt.Sprintf("%s.%s.%s.%s",
		id.Tags["project"],
		id.Tags["family"],
		id.Tags["group"],
		id.Name,
	)
}

// ParseModelIdWithPattern parses a string into a ModelId using the given pattern.
// This is a lower-level API intended for tests and special cases where a custom
// pattern is needed.
//
// For production code, prefer ParseCanonicalModelId with proper config.
func ParseModelIdWithPattern(pattern, s string) (ModelId, error) {
	return modelIdFromStringWithPattern(pattern, s)
}
