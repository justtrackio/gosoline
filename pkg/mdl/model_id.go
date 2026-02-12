package mdl

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/spf13/cast"
)

// Placeholder constants for ModelId patterns.
const (
	delimiterDot       = "."
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
	Get(key string, optionalDefault ...any) (any, error)
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
//
// After loading the domain pattern, this method validates that all {app.tags.<key>}
// placeholders referenced in the pattern have corresponding entries in the merged
// Tags map. This prevents formatDomain from silently replacing missing tags with
// empty strings, which would produce malformed canonical IDs.
func (m *ModelId) PadFromConfig(config ConfigProvider) error {
	if err := m.padEnvFromConfig(config); err != nil {
		return err
	}

	if err := m.padAppFromConfig(config); err != nil {
		return err
	}

	if err := m.mergeTagsFromConfig(config); err != nil {
		return err
	}

	if err := m.loadDomainPatternFromConfig(config); err != nil {
		return err
	}

	if err := validatePatternTagsPresent(m.DomainPattern, m.Tags); err != nil {
		return fmt.Errorf("invalid %s: %w", ConfigKeyModelIdDomainPattern, err)
	}

	return nil
}

func (m *ModelId) padEnvFromConfig(config ConfigProvider) error {
	var err error

	if m.Env != "" {
		return nil
	}

	if m.Env, err = config.GetString("app.env"); err != nil {
		return fmt.Errorf("failed to read app.env from config: %w", err)
	}

	m.Env = strings.TrimSpace(m.Env)

	if m.Env == "" {
		return fmt.Errorf("environment (app.env) is required to be not empty")
	}

	return nil
}

func (m *ModelId) padAppFromConfig(config ConfigProvider) error {
	var err error

	if m.App != "" {
		return nil
	}

	if m.App, err = config.GetString("app.name"); err != nil {
		return fmt.Errorf("failed to read app.name from config: %w", err)
	}

	m.App = strings.TrimSpace(m.App)

	if m.App == "" {
		return fmt.Errorf("app name (app.name) is required to be not empty")
	}

	return nil
}

func (m *ModelId) mergeTagsFromConfig(config ConfigProvider) error {
	var err error
	var configTags map[string]any

	if configTags, err = config.GetStringMap("app.tags"); err != nil {
		configTags = make(map[string]any)
	}

	if m.Tags == nil {
		m.Tags = make(map[string]string)
	}

	for k, v := range configTags {
		if _, exists := m.Tags[k]; exists {
			continue
		}

		if m.Tags[k], err = cast.ToStringE(v); err != nil {
			return fmt.Errorf("failed to cast app.tags.%s to string: %w", k, err)
		}
	}

	return nil
}

func (m *ModelId) loadDomainPatternFromConfig(config ConfigProvider) error {
	if m.DomainPattern != "" {
		return nil
	}

	var err error
	var val any
	var domainPattern string

	if val, err = config.Get(ConfigKeyModelIdDomainPattern); err != nil {
		return fmt.Errorf("failed to read %s from config: %w", ConfigKeyModelIdDomainPattern, err)
	}

	if domainPattern, err = cast.ToStringE(val); err != nil {
		return fmt.Errorf("failed to cast %s to string: %w", ConfigKeyModelIdDomainPattern, err)
	}

	if domainPattern == "" {
		return fmt.Errorf("model id domain pattern is empty")
	}

	m.DomainPattern = domainPattern

	return nil
}

// validateModelIdDomainPattern checks that a pattern is valid and parseable.
//
// A valid pattern:
//   - is a non-empty string
//   - if it contains placeholders, they must be recognized placeholders
//   - static text may appear anywhere in the pattern (prefixes, suffixes, delimiters between placeholders)
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
	for _, ph := range placeholders {
		switch {
		case ph == PlaceholderAppEnv:
		case ph == PlaceholderAppName:
			continue

		case strings.HasPrefix(ph, PlaceholderAppTags):
			tagKey := strings.TrimPrefix(ph, PlaceholderAppTags)
			if tagKey == "" {
				return fmt.Errorf("tag key is empty for placeholder {%s}", ph)
			}
		default:
			return fmt.Errorf("unknown placeholder {%s} in domain pattern", ph)
		}
	}

	return nil
}

// extractPlaceholders returns all placeholder names from a pattern.
// For "{app.tags.project}.{modelId}", returns ["app.tags.project", "modelId"].
func extractPlaceholders(pattern string) []string {
	matches := domainRegex.FindAllStringSubmatch(pattern, -1)
	placeholders := make([]string, 0, len(matches))

	for _, match := range matches {
		if len(match) > 1 && match[1] != "" {
			placeholders = append(placeholders, match[1])
		}
	}

	return placeholders
}

// validatePatternTagsPresent checks that all {app.tags.<key>} placeholders
// referenced in the domain pattern have corresponding entries in the tags map.
// This prevents formatDomain from silently replacing missing tags with empty strings.
func validatePatternTagsPresent(domainPattern string, tags map[string]string) error {
	placeholders := extractPlaceholders(domainPattern)

	for _, ph := range placeholders {
		if !strings.HasPrefix(ph, PlaceholderAppTags) {
			continue
		}

		tagKey := strings.TrimPrefix(ph, PlaceholderAppTags)
		if _, exists := tags[tagKey]; !exists {
			return fmt.Errorf("tag %q is required by domain pattern %q but not set in tags", tagKey, domainPattern)
		}
	}

	return nil
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
