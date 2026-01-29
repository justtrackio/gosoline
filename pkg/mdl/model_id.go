package mdl

import (
	"fmt"
	"strings"
)

// Placeholder constants for ModelId patterns.
const (
	delimiterDot       = "."
	PlaceholderModelId = "modelId"
	PlaceholderAppEnv  = "app.env"
	PlaceholderAppName = "app.name"
	PlaceholderAppTags = "app.tags."
)

// ConfigKeyModelIdPattern is the config key for the required model id pattern.
const ConfigKeyModelIdPattern = "app.model_id.pattern"

// ConfigProvider is an interface for reading config values.
// It is implemented by cfg.Config.
type ConfigProvider interface {
	GetString(key string, optionalDefault ...string) (string, error)
	GetStringMap(key string, optionalDefault ...map[string]any) (map[string]any, error)
}

// ModelId represents the identity of a model using dynamic tags.
//
// Unlike the legacy fixed-field ModelId (Project/Family/Group/etc),
// this version uses a dynamic tags map that can hold any key-value pairs.
//
// The canonical string form is determined by the app.model_id.pattern config key.
// Call PadFromConfig() once to populate fields from config (including the pattern),
// then use Format() to obtain the canonical string representation.
// Use DebugModelIdString() for logging/debugging purposes.
type ModelId struct {
	// Name is the model's name (the {modelId} placeholder)
	Name string `cfg:"name"`
	// Env is the environment (the {app.env} placeholder)
	Env string `cfg:"env"`
	// App is the application name (the {app.name} placeholder)
	App string `cfg:"app"`
	// Tags holds dynamic tag values (for {app.tags.<key>} placeholders)
	Tags map[string]string `cfg:"tags"`

	// pattern is the formatting pattern read from config during PadFromConfig.
	// This is private and not serialized; it is set by PadFromConfig.
	pattern string
}

// format expands placeholders in the given pattern using ModelId values.
// This is an internal method - call PadFromConfig once, then use Format() for public API.
//
// Supported placeholders:
//   - {modelId} - the model's Name
//   - {app.env} - the Env field
//   - {app.name} - the App field
//   - {app.tags.<key>} - any tag from the Tags map
//
// If the pattern contains no placeholders, it is returned as-is.
// This allows for explicit table name overrides.
//
// Returns an error if:
//   - the pattern contains unknown placeholders
//   - a required tag is missing (referenced in pattern but not in Tags)
//   - {app.env} or {app.name} is referenced but the field is empty
func (m *ModelId) format(pattern string) (string, error) {
	if err := validateModelIdPattern(pattern); err != nil {
		return "", err
	}

	result := pattern
	var missingTags []string

	// Extract and process all placeholders
	placeholders := extractPlaceholders(pattern)
	for _, ph := range placeholders {
		var value string
		var ok bool

		switch {
		case ph == PlaceholderModelId:
			value = m.Name
			ok = true
		case ph == PlaceholderAppEnv:
			value = m.Env
			ok = m.Env != ""
			if !ok {
				return "", fmt.Errorf("pattern requires %s but it is empty", PlaceholderAppEnv)
			}
		case ph == PlaceholderAppName:
			value = m.App
			ok = m.App != ""
			if !ok {
				return "", fmt.Errorf("pattern requires %s but it is empty", PlaceholderAppName)
			}
		case strings.HasPrefix(ph, PlaceholderAppTags):
			tagKey := strings.TrimPrefix(ph, PlaceholderAppTags)
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

	return result, nil
}

// PadFromConfig fills in empty fields of ModelId from config.
//
// This method reads:
//   - app.env -> Env (if empty)
//   - app.name -> App (if empty)
//   - app.tags.* -> Tags (merged, existing tags take precedence)
//   - app.model_id.pattern -> pattern (if empty and available)
//
// The pattern is read if available, enabling Format() to work afterward.
// If the pattern config key is missing, the pattern field is left empty,
// and Format() will return an error when called.
//
// All identity fields (env, app, tags) are optional. If a config key is not found,
// the corresponding field is left unchanged. This allows patterns to determine
// what's required (enforced at format time).
func (m *ModelId) PadFromConfig(config ConfigProvider) error {
	m.padEnvFromConfig(config)
	m.padAppFromConfig(config)
	m.mergeTagsFromConfig(config)

	return m.loadPatternFromConfig(config)
}

func (m *ModelId) padEnvFromConfig(config ConfigProvider) {
	if m.Env != "" {
		return
	}

	if env, err := config.GetString("app.env"); err == nil {
		m.Env = env
	}
}

func (m *ModelId) padAppFromConfig(config ConfigProvider) {
	if m.App != "" {
		return
	}

	if app, err := config.GetString("app.name"); err == nil {
		m.App = app
	}
}

func (m *ModelId) mergeTagsFromConfig(config ConfigProvider) {
	configTags, err := config.GetStringMap("app.tags")
	if err != nil {
		configTags = make(map[string]any)
	}

	if m.Tags == nil {
		m.Tags = make(map[string]string)
	}

	for k, v := range configTags {
		if _, exists := m.Tags[k]; exists {
			continue
		}

		if strVal, ok := v.(string); ok {
			m.Tags[k] = strVal
		}
	}
}

func (m *ModelId) loadPatternFromConfig(config ConfigProvider) error {
	if m.pattern != "" {
		return nil
	}

	pattern, err := config.GetString(ConfigKeyModelIdPattern)
	if err != nil {
		// If app.model_id.pattern is not in config, leave pattern empty - Format() will error when called
		return nil
	}

	if pattern == "" {
		return fmt.Errorf("model id pattern is empty")
	}

	if err := validateModelIdPattern(pattern); err != nil {
		return fmt.Errorf("invalid %s: %w", ConfigKeyModelIdPattern, err)
	}

	m.pattern = pattern

	return nil
}

// Format returns the canonical string representation of the ModelId.
//
// This method requires that the pattern has been set via PadFromConfig.
// If no pattern is set, Format returns an error.
//
// Returns an error if:
//   - the pattern is not set (call PadFromConfig first)
//   - the pattern references fields/tags that are missing from the ModelId
func (m ModelId) Format() (string, error) {
	if m.pattern == "" {
		return "", fmt.Errorf("model id pattern is not set; call PadFromConfig first")
	}

	result, err := m.format(m.pattern)
	if err != nil {
		return "", fmt.Errorf("failed to format model id with pattern %q: %w", m.pattern, err)
	}

	return result, nil
}

// modelIdFromStringWithPattern parses a string into a ModelId using the given pattern.
// This is an internal function - use ParseModelId() for public API.
//
// The pattern must be "parseable" - consisting only of placeholders separated
// by a single dot "." delimiter. For example:
//   - "{app.tags.project}.{app.tags.family}.{app.tags.group}.{modelId}"
//   - "{app.env}.{modelId}"
//
// The string is split by the delimiter, and each segment is mapped to the
// corresponding placeholder in the pattern.
func modelIdFromStringWithPattern(pattern, str string) (ModelId, error) {
	if err := validateModelIdPattern(pattern); err != nil {
		return ModelId{}, fmt.Errorf("invalid pattern: %w", err)
	}

	placeholders := extractPlaceholders(pattern)
	parts := strings.Split(str, delimiterDot)

	if len(parts) != len(placeholders) {
		return ModelId{}, fmt.Errorf(
			"string %q has %d segments but pattern expects %d (pattern: %s, delimiter: %q)",
			str, len(parts), len(placeholders), pattern, delimiterDot,
		)
	}

	modelId := ModelId{
		Tags: make(map[string]string),
	}

	for i, ph := range placeholders {
		value := parts[i]

		switch {
		case ph == PlaceholderModelId:
			modelId.Name = value
		case ph == PlaceholderAppEnv:
			modelId.Env = value
		case ph == PlaceholderAppName:
			modelId.App = value
		case strings.HasPrefix(ph, PlaceholderAppTags):
			tagKey := strings.TrimPrefix(ph, PlaceholderAppTags)
			modelId.Tags[tagKey] = value
		}
	}

	return modelId, nil
}

// validateModelIdPattern checks that a pattern is valid and parseable.
//
// A valid pattern:
//   - is a non-empty string
//   - if it contains placeholders, they must be recognized placeholders
//   - for multiple placeholders, they must be separated by a single dot "." delimiter
//   - no static text between placeholders (except the dot delimiter)
//
// A pattern with no placeholders is valid and will be returned as-is.
func validateModelIdPattern(pattern string) error {
	if pattern == "" {
		return fmt.Errorf("pattern cannot be empty")
	}

	placeholders := extractPlaceholders(pattern)
	if len(placeholders) == 0 {
		// No placeholders - this is a static pattern, which is valid
		// (useful for explicit table name overrides)
		return nil
	}

	// Validate each placeholder
	for _, ph := range placeholders {
		if !isAllowedModelIdPlaceholder(ph) {
			return fmt.Errorf("unknown placeholder {%s} in pattern", ph)
		}
	}

	// Check that pattern is parseable (placeholders + dot delimiter only)
	if len(placeholders) == 1 {
		// Single placeholder - must be the entire pattern
		if pattern != "{"+placeholders[0]+"}" {
			return fmt.Errorf("pattern contains static text which makes it unparseable")
		}

		return nil
	}

	// Multiple placeholders - ensure they are joined by dot
	var expectedParts []string
	for _, ph := range placeholders {
		expectedParts = append(expectedParts, "{"+ph+"}")
	}
	expectedPattern := strings.Join(expectedParts, ".")

	if pattern != expectedPattern {
		return fmt.Errorf("pattern must consist of placeholders separated by dots (.), got %q", pattern)
	}

	return nil
}

// extractPlaceholders returns all placeholder names from a pattern.
// For "{app.tags.project}.{modelId}", returns ["app.tags.project", "modelId"].
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

// isAllowedModelIdPlaceholder checks if a placeholder name is valid for ModelId patterns.
func isAllowedModelIdPlaceholder(placeholder string) bool {
	switch {
	case placeholder == PlaceholderModelId:
		return true
	case placeholder == PlaceholderAppEnv:
		return true
	case placeholder == PlaceholderAppName:
		return true
	case strings.HasPrefix(placeholder, PlaceholderAppTags):
		// Ensure there's actually a tag key after the prefix
		tagKey := strings.TrimPrefix(placeholder, PlaceholderAppTags)

		return tagKey != ""
	default:
		return false
	}
}

type Identifiable interface {
	GetId() *uint
}

type Keyed interface {
	GetKey() string
}

type Identifier struct {
	Id *uint `json:"id" binding:"required"`
}

func (i *Identifier) GetId() *uint {
	if i == nil {
		return nil
	}

	return i.Id
}

func WithIdentifier(id *uint) *Identifier {
	return &Identifier{
		Id: id,
	}
}

func UuidWithDashes(uuid *string) (*string, error) {
	if uuid == nil {
		return nil, fmt.Errorf("the uuid should not be nil")
	}

	if strings.Contains(*uuid, "-") {
		return uuid, nil
	}

	if len(*uuid) != 32 {
		return uuid, fmt.Errorf("the uuid should be exactly 32 bytes long, but was: %d", len(*uuid))
	}

	dashed := fmt.Sprintf("%s-%s-%s-%s-%s", (*uuid)[0:8], (*uuid)[8:12], (*uuid)[12:16], (*uuid)[16:20], (*uuid)[20:32])

	return &dashed, nil
}

type Resource interface {
	GetResourceName() string
}

type ResourceContextAware interface {
	GetResourceContext() map[string]any
}
