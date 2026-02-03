package mdl

import (
	"fmt"
	"strings"
)

// ParseModelId parses a string into a ModelId using the required domain pattern from config.
// This is the only supported way to parse a canonical model id string.
//
// Returns an error if:
//   - the domain pattern is not configured or invalid
//   - the string doesn't match the expected domain pattern structure
func ParseModelId(config ConfigProvider, s string) (ModelId, error) {
	var err error
	var domainPattern string
	var id ModelId

	if domainPattern, err = config.GetString(ConfigKeyModelIdDomainPattern); err != nil {
		return ModelId{}, fmt.Errorf("%s must be set: %w", ConfigKeyModelIdDomainPattern, err)
	}

	if domainPattern == "" {
		return ModelId{}, fmt.Errorf("%s must not be empty", ConfigKeyModelIdDomainPattern)
	}

	if err = validateModelIdDomainPattern(domainPattern); err != nil {
		return ModelId{}, fmt.Errorf("invalid %s: %w", ConfigKeyModelIdDomainPattern, err)
	}

	if id, err = modelIdFromStringWithDomainPattern(domainPattern, s); err != nil {
		return ModelId{}, fmt.Errorf("failed to parse model id %q with domain pattern %q: %w", s, domainPattern, err)
	}

	id.DomainPattern = domainPattern

	return id, nil
}

// modelIdFromStringWithDomainPattern parses a string into a ModelId using the given domain pattern.
// This is an internal function - use ParseModelId() for public API.
//
// The pattern must be "parseable" - consisting only of placeholders separated
// by a single dot "." delimiter. For example:
//   - "{app.tags.project}.{app.tags.family}.{app.tags.group}"
//   - "{app.env}"
//
// The string is split by the delimiter, and each segment is mapped to the
// corresponding placeholder in the pattern. The last segment is implicitly
// treated as the model name.
func modelIdFromStringWithDomainPattern(domainPattern, str string) (ModelId, error) {
	if err := validateModelIdDomainPattern(domainPattern); err != nil {
		return ModelId{}, fmt.Errorf("invalid pattern: %w", err)
	}

	placeholders := extractPlaceholders(domainPattern)
	parts := strings.Split(str, delimiterDot)

	expectedParts := len(placeholders) + 1
	if len(parts) != expectedParts {
		return ModelId{}, fmt.Errorf(
			"string %q has %d segments but pattern expects %d (%d from pattern + 1 for model name)",
			str, len(parts), expectedParts, len(placeholders),
		)
	}

	modelId := ModelId{
		Tags: make(map[string]string),
	}

	for i, ph := range placeholders {
		value := parts[i]

		switch {
		case ph == PlaceholderAppEnv:
			modelId.Env = value
		case ph == PlaceholderAppName:
			modelId.App = value
		case strings.HasPrefix(ph, PlaceholderAppTags):
			tagKey := strings.TrimPrefix(ph, PlaceholderAppTags)
			modelId.Tags[tagKey] = value
		}
	}

	modelId.Name = parts[len(parts)-1]

	return modelId, nil
}
