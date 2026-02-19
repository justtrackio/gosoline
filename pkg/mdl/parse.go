package mdl

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/spf13/cast"
)

// ParseModelId parses a string into a ModelId using the required domain pattern from config.
// This is the only supported way to parse a canonical model id string.
//
// Returns an error if:
//   - the domain pattern is not configured or invalid
//   - the string doesn't match the expected domain pattern structure
func ParseModelId(config ConfigProvider, s string) (ModelId, error) {
	var err error
	var val any
	var domainPattern string
	var id ModelId

	if val, err = config.Get(ConfigKeyModelIdDomainPattern); err != nil {
		return ModelId{}, fmt.Errorf("%s must be set: %w", ConfigKeyModelIdDomainPattern, err)
	}

	if domainPattern, err = cast.ToStringE(val); err != nil {
		return ModelId{}, fmt.Errorf("failed to cast %s to string: %w", ConfigKeyModelIdDomainPattern, err)
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
// The parser builds a regex from the domain pattern by:
//   - escaping all static text parts
//   - replacing each placeholder with a named capture group
//   - appending a capture group for the model name (after the final dot)
//
// The parser supports mixed patterns containing both static text and placeholders.
// For example:
//   - "{app.tags.project}.{app.tags.family}.{app.tags.group}" (dot-separated placeholders)
//   - "prefix-{app.env}" (static prefix with placeholder)
//   - "{app.tags.project}-{app.env}" (non-dot delimiter between placeholders)
//   - "my-{app.tags.project}.{app.env}-suffix" (mixed static text and placeholders)
func modelIdFromStringWithDomainPattern(domainPattern, str string) (ModelId, error) {
	if err := validateModelIdDomainPattern(domainPattern); err != nil {
		return ModelId{}, fmt.Errorf("invalid pattern: %w", err)
	}

	parseRegex, groupNames, err := buildParseRegex(domainPattern)
	if err != nil {
		return ModelId{}, fmt.Errorf("failed to build parse regex for pattern %q: %w", domainPattern, err)
	}

	match := parseRegex.FindStringSubmatch(str)
	if match == nil {
		return ModelId{}, fmt.Errorf(
			"string %q does not match domain pattern %q (regex: %s)",
			str, domainPattern, parseRegex.String(),
		)
	}

	modelId := ModelId{
		Tags: make(map[string]string),
	}

	for i, name := range groupNames {
		if i+1 >= len(match) {
			break
		}

		value := match[i+1]

		switch {
		case name == placeholderGroupModelName:
			modelId.Name = value
		case name == PlaceholderAppEnv:
			modelId.Env = value
		case name == PlaceholderAppName:
			modelId.App = value
		case strings.HasPrefix(name, PlaceholderAppTags):
			tagKey := strings.TrimPrefix(name, PlaceholderAppTags)
			modelId.Tags[tagKey] = value
		}
	}

	return modelId, nil
}

// placeholderGroupModelName is the internal group name used for the model name capture group.
const placeholderGroupModelName = "__model_name__"

// staticParseRegex matches the entire input as the model name (used for static patterns with no placeholders).
var staticParseRegex = regexp.MustCompile(`^(.+)$`)

// buildParseRegex builds a regex from a domain pattern for parsing model id strings.
//
// Each placeholder in the pattern becomes a capture group matching non-dot characters.
// A final capture group is appended to match the model name after the last dot.
//
// For static patterns (no placeholders), the regex matches only the model name,
// since the static domain part is not included in the formatted string's parse input
// (i.e., the model name is the entire input).
//
// Returns the compiled regex and an ordered list of group names (placeholder keys)
// corresponding to each capture group.
func buildParseRegex(domainPattern string) (*regexp.Regexp, []string, error) {
	placeholders := extractPlaceholders(domainPattern)

	// Static pattern (no placeholders): the entire input is just the model name.
	if len(placeholders) == 0 {
		return staticParseRegex, []string{placeholderGroupModelName}, nil
	}

	// Build the regex by replacing placeholders with capture groups.
	// We process the pattern left-to-right, escaping static parts and inserting groups.
	regexStr := ""
	remaining := domainPattern
	var groupNames []string

	for _, ph := range placeholders {
		placeholder := "{" + ph + "}"
		idx := strings.Index(remaining, placeholder)

		if idx < 0 {
			return nil, nil, fmt.Errorf("placeholder {%s} not found in remaining pattern %q", ph, remaining)
		}

		// Escape the static text before this placeholder
		if idx > 0 {
			regexStr += regexp.QuoteMeta(remaining[:idx])
		}

		// Add a capture group for this placeholder.
		// Use [^.]+ to match non-dot characters (greedy within the segment).
		regexStr += "([^.]+)"
		groupNames = append(groupNames, ph)

		remaining = remaining[idx+len(placeholder):]
	}

	// Append any remaining static text after the last placeholder
	if remaining != "" {
		regexStr += regexp.QuoteMeta(remaining)
	}

	// Append the model name capture: a dot followed by the rest of the string
	regexStr += `\.(.+)`
	groupNames = append(groupNames, placeholderGroupModelName)

	// Anchor the regex
	regexStr = "^" + regexStr + "$"

	compiled, err := regexp.Compile(regexStr)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to compile regex %q: %w", regexStr, err)
	}

	return compiled, groupNames, nil
}
