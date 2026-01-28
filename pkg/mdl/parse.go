package mdl

import (
	"fmt"
)

// ParseModelId parses a string into a ModelId using the required pattern from config.
// This is the only supported way to parse a canonical model id string.
//
// Returns an error if:
//   - the pattern is not configured or invalid
//   - the string doesn't match the expected pattern structure
func ParseModelId(config ConfigProvider, s string) (ModelId, error) {
	pattern, err := readModelIdPattern(config)
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

// ParseModelIdWithPattern parses a string into a ModelId using the given pattern.
// This is a lower-level API intended for tests and special cases where a custom
// pattern is needed.
//
// For production code, prefer ParseModelId with proper config.
func ParseModelIdWithPattern(pattern, s string) (ModelId, error) {
	return modelIdFromStringWithPattern(pattern, s)
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

func readModelIdPattern(config ConfigProvider) (string, error) {
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
