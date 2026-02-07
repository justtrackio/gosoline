package mdl

import (
	"fmt"
	"regexp"
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

var domainRegex = regexp.MustCompile(`(?m)\{([^\}]+)\}`)

// ConfigKeyModelIdDomainPattern is the config key for the required model id domain pattern.
const ConfigKeyModelIdDomainPattern = "app.model_id.domain_pattern"

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
// The canonical string form is determined by the app.model_id.domain_pattern config key.
// Call PadFromConfig() once to populate fields from config (including the domain pattern),
// then use String() to obtain the canonical string representation.
type ModelId struct {
	// Name is the model's name (the {modelId} placeholder)
	Name string `cfg:"name"`
	// Env is the environment (the {app.env} placeholder)
	Env string `cfg:"env"`
	// App is the application name (the {app.name} placeholder)
	App string `cfg:"app"`
	// Tags holds dynamic tag values (for {app.tags.<key>} placeholders)
	Tags map[string]string `cfg:"tags"`

	// DomainPattern is the formatting pattern read from config during PadFromConfig.
	// This is private and not serialized; it is set by PadFromConfig.
	DomainPattern string
}

func (i ModelId) ToMap() map[string]string {
	mss := map[string]string{
		"name":     i.Name,
		"app.name": i.App,
		"app.env":  i.Env,
	}

	for key, value := range i.Tags {
		mss[fmt.Sprintf("app.tags.%s", key)] = value
	}

	return mss
}

// format expands placeholders in the given pattern using ModelId values.
// This is an internal method - call PadFromConfig once, then use String() for public API.
//
// Supported placeholders:
//   - {app.env} - the Env field
//   - {app.name} - the App field
//   - {app.tags.<key>} - any tag from the Tags map
//
// The model name (Name field) is automatically appended as the last segment.
//
// If the pattern contains no placeholders, it is returned as-is (with the model name appended).
//
// Returns an error if:
//   - the pattern contains unknown placeholders
//   - the pattern contains {modelId} (it is no longer allowed)
//   - the model name is empty
//   - a required tag is missing (referenced in pattern but not in Tags)
//   - {app.env} or {app.name} is referenced but the field is empty
func (m *ModelId) format(domainPattern string) string {
	domain := m.formatDomain(domainPattern)

	return fmt.Sprintf("%s.%s", domain, m.Name)
}

func (m *ModelId) formatDomain(domainPattern string) string {
	var value string

	result := domainPattern
	matches := domainRegex.FindAllStringSubmatch(domainPattern, -1)

	for _, match := range matches {
		placeholder := match[1]

		switch {
		case placeholder == PlaceholderAppEnv:
			value = m.Env
		case placeholder == PlaceholderAppName:
			value = m.App
		case strings.HasPrefix(placeholder, PlaceholderAppTags):
			tagKey := strings.TrimPrefix(placeholder, PlaceholderAppTags)
			value = m.Tags[tagKey]
		default:
			continue
		}

		result = strings.ReplaceAll(result, match[0], value)
	}

	return result
}

// String returns the canonical string representation of the ModelId.
//
// This method requires that the DomainPattern has been set via PadFromConfig.
// If no DomainPattern is set, String returns an error.
//
// Returns an error if:
//   - the DomainPattern is not set (call PadFromConfig first)
//   - the DomainPattern references fields/tags that are missing from the ModelId
func (m ModelId) String() string {
	return m.format(m.DomainPattern)
}

// DomainString returns the canonical domain string representation of the ModelId (without the model name).
//
// This method requires that the DomainPattern has been set via PadFromConfig.
// If no DomainPattern is set, DomainString returns an error.
//
// Returns an error if:
//   - the DomainPattern is not set (call PadFromConfig first)
//   - the DomainPattern references fields/tags that are missing from the ModelId
func (m ModelId) DomainString() string {
	return m.formatDomain(m.DomainPattern)
}

// PadFromConfig fills in empty fields of ModelId from config.
//
// This method reads:
//   - app.env -> Env (if empty)
//   - app.name -> App (if empty)
//   - app.tags.* -> Tags (merged, existing tags take precedence)
//   - app.model_id.domain_pattern -> DomainPattern (if empty and available)
//
// The DomainPattern is read if available, enabling String() to work afterward.
// If the domain pattern config key is missing, the DomainPattern field is left empty,
// and String() will return an error when called.
//
// All identity fields (env, app, tags) are optional. If a config key is not found,
// the corresponding field is left unchanged. This allows patterns to determine
// what's required (enforced at format time).
func (m *ModelId) PadFromConfig(config ConfigProvider) error {
	m.padEnvFromConfig(config)
	m.padAppFromConfig(config)
	m.mergeTagsFromConfig(config)

	return m.loadDomainPatternFromConfig(config)
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

func (m *ModelId) loadDomainPatternFromConfig(config ConfigProvider) error {
	if m.DomainPattern != "" {
		return nil
	}

	domainPattern, err := config.GetString(ConfigKeyModelIdDomainPattern)
	if err != nil {
		// If app.model_id.domain_pattern is not in config, leave DomainPattern empty - String() will error when called
		return nil
	}

	if domainPattern == "" {
		return fmt.Errorf("model id domain pattern is empty")
	}

	if err := validateModelIdDomainPattern(domainPattern); err != nil {
		return fmt.Errorf("invalid %s: %w", ConfigKeyModelIdDomainPattern, err)
	}

	m.DomainPattern = domainPattern

	return nil
}

// validateModelIdDomainPattern checks that a pattern is valid and parseable.
//
// A valid pattern:
//   - is a non-empty string
//   - if it contains placeholders, they must be recognized placeholders
//   - for multiple placeholders, they must be separated by a single dot "." delimiter
//   - no static text between placeholders (except the dot delimiter)
//
// A pattern with no placeholders is valid and will be returned as-is.
func validateModelIdDomainPattern(domainPattern string) error {
	if domainPattern == "" {
		return fmt.Errorf("pattern cannot be empty")
	}

	placeholders := extractPlaceholders(domainPattern)
	if len(placeholders) == 0 {
		// No placeholders - this is a static pattern, which is valid
		// (useful for explicit table name overrides)
		return nil
	}

	// Validate each placeholder
	if err := validatePlaceholders(placeholders); err != nil {
		return err
	}

	// Check that pattern is parseable (placeholders + dot delimiter only)
	return validatePatternFormat(domainPattern, placeholders)
}

func validatePlaceholders(placeholders []string) error {
	for _, ph := range placeholders {
		if ph == PlaceholderModelId {
			return fmt.Errorf("{modelId} placeholder is no longer allowed in patterns; the model name is automatically appended as the last segment")
		}

		if !isAllowedModelIdPlaceholder(ph) {
			return fmt.Errorf("unknown placeholder {%s} in pattern", ph)
		}
	}

	return nil
}

func validatePatternFormat(domainPattern string, placeholders []string) error {
	if len(placeholders) == 1 {
		// Single placeholder - must be the entire pattern
		if domainPattern != "{"+placeholders[0]+"}" {
			return fmt.Errorf("pattern contains static text which makes it unparseable")
		}

		return nil
	}

	// Multiple placeholders - ensure they are joined by dot
	expectedPattern := buildExpectedPattern(placeholders)

	if domainPattern != expectedPattern {
		return validateDelimiterError(domainPattern, expectedPattern, placeholders)
	}

	return nil
}

func buildExpectedPattern(placeholders []string) string {
	var expectedParts []string
	for _, ph := range placeholders {
		expectedParts = append(expectedParts, "{"+ph+"}")
	}

	return strings.Join(expectedParts, ".")
}

func validateDelimiterError(domainPattern, expectedPattern string, placeholders []string) error {
	// Provide a more specific error message if the only difference is the delimiter
	if len(domainPattern) == len(expectedPattern) && hasMatchingPlaceholders(domainPattern, placeholders) {
		return fmt.Errorf("pattern must consist of placeholders separated by dots (.), got %q", domainPattern)
	}

	return fmt.Errorf("pattern contains static text which makes it unparseable")
}

func hasMatchingPlaceholders(domainPattern string, placeholders []string) bool {
	currentIdx := 0
	for _, ph := range placeholders {
		part := "{" + ph + "}"
		idx := strings.Index(domainPattern[currentIdx:], part)
		if idx == -1 {
			return false
		}
		currentIdx += idx + len(part)
	}

	return true
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
