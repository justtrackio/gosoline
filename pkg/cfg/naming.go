package cfg

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// NamingTemplate provides strict placeholder validation and expansion
// for resource naming patterns like SQS queues, SNS topics, etc.
//
// Supported identity placeholders:
//   - {app.env} - environment
//   - {app.name} - application name
//   - {app.tags.<key>} - any tag from identity (e.g., {app.tags.project}, {app.tags.region})
//
// Components can register additional resource-specific placeholders
// (e.g., {queueId}, {topicId}, {streamName}).
type NamingTemplate struct {
	pattern              string
	resourcePlaceholders map[string]struct{}
	resourceValues       map[string]string
}

// placeholderRegex matches {placeholder} tokens in naming patterns.
// It captures the placeholder name without braces.
var placeholderRegex = regexp.MustCompile(`\{([^}]*)\}`)

// tagPlaceholderPrefix is the prefix for dynamic tag placeholders.
const tagPlaceholderPrefix = "app.tags."

// NewNamingTemplate creates a new template with the given pattern and
// additional resource-specific placeholders (beyond identity placeholders).
//
// Identity placeholders ({app.env}, {app.name}, {app.tags.*}) are always allowed.
// Resource placeholders must be explicitly registered.
//
// Example:
//
//	tmpl := cfg.NewNamingTemplate(pattern, "queueId")
func NewNamingTemplate(pattern string, resourcePlaceholders ...string) *NamingTemplate {
	rp := make(map[string]struct{})
	for _, p := range resourcePlaceholders {
		rp[p] = struct{}{}
	}

	return &NamingTemplate{
		pattern:              pattern,
		resourcePlaceholders: rp,
		resourceValues:       make(map[string]string),
	}
}

// WithResourceValue sets a resource-specific placeholder value.
// This is used for placeholders like {queueId}, {topicId}, etc.
//
// Example:
//
//	tmpl.WithResourceValue("queueId", "my-queue")
func (t *NamingTemplate) WithResourceValue(key, value string) *NamingTemplate {
	t.resourceValues[key] = value

	return t
}

// Validate checks the pattern for:
//   - Unclosed placeholders (e.g., "{foo" without "}")
//   - Empty placeholders ({})
//   - Empty tag key ({app.tags.})
//   - Unknown placeholders not in the allowed set
//
// Returns an error describing the issue, or nil if valid.
func (t *NamingTemplate) Validate() error {
	// Check for unclosed braces
	openCount := strings.Count(t.pattern, "{")
	closeCount := strings.Count(t.pattern, "}")

	if openCount != closeCount {
		return fmt.Errorf("unclosed placeholder in pattern %q", t.pattern)
	}

	// Find all placeholders and check against allowlist
	matches := placeholderRegex.FindAllStringSubmatch(t.pattern, -1)
	var unknown []string

	for _, match := range matches {
		placeholder := match[1]

		// Check for empty placeholder
		if placeholder == "" {
			return fmt.Errorf("empty placeholder {} in pattern %q", t.pattern)
		}

		// Check if it's an allowed placeholder
		if !t.isAllowedPlaceholder(placeholder) {
			unknown = append(unknown, placeholder)
		}
	}

	if len(unknown) > 0 {
		slices.Sort(unknown)

		return fmt.Errorf("unknown placeholder(s) {%s} in pattern %q", strings.Join(unknown, "}, {"), t.pattern)
	}

	return nil
}

// isAllowedPlaceholder checks if a placeholder is allowed.
// Allowed placeholders are:
//   - "app.env" and "app.name" (fixed identity fields)
//   - any "app.tags.<key>" where <key> is non-empty
//   - any registered resource placeholder
func (t *NamingTemplate) isAllowedPlaceholder(placeholder string) bool {
	// Fixed identity placeholders
	if placeholder == "app.env" || placeholder == "app.name" {
		return true
	}

	// Dynamic tag placeholders: app.tags.<key> where <key> is non-empty
	if strings.HasPrefix(placeholder, tagPlaceholderPrefix) {
		tagKey := strings.TrimPrefix(placeholder, tagPlaceholderPrefix)

		return tagKey != "" // reject {app.tags.} (empty key)
	}

	// Resource-specific placeholders
	_, ok := t.resourcePlaceholders[placeholder]

	return ok
}

// RequiredTags returns the tag keys that are required based on the pattern.
// For example, if the pattern contains {app.tags.project}, "project" is required.
// If the pattern contains {app.tags.region}, "region" is required.
func (t *NamingTemplate) RequiredTags() []string {
	var required []string

	matches := placeholderRegex.FindAllStringSubmatch(t.pattern, -1)
	seen := make(map[string]struct{})

	for _, match := range matches {
		placeholder := match[1]
		if strings.HasPrefix(placeholder, tagPlaceholderPrefix) {
			tag := strings.TrimPrefix(placeholder, tagPlaceholderPrefix)
			if tag != "" { // skip malformed {app.tags.}
				if _, ok := seen[tag]; !ok {
					required = append(required, tag)
					seen[tag] = struct{}{}
				}
			}
		}
	}

	return required
}

// RequiresAppName returns true if the pattern contains {app.name}.
func (t *NamingTemplate) RequiresAppName() bool {
	return strings.Contains(t.pattern, "{app.name}")
}

// RequiresEnv returns true if the pattern contains {app.env}.
func (t *NamingTemplate) RequiresEnv() bool {
	return strings.Contains(t.pattern, "{app.env}")
}

// Expand replaces all placeholders with their values from the identity
// and resource values. This should only be called after Validate().
//
// The function does NOT validate - call Validate() first if needed.
func (t *NamingTemplate) Expand(identity AppIdentity) string {
	result := t.pattern

	// Replace fixed identity placeholders
	result = strings.ReplaceAll(result, "{app.env}", identity.Env)
	result = strings.ReplaceAll(result, "{app.name}", identity.Name)

	// Replace resource placeholders
	for k, v := range t.resourceValues {
		result = strings.ReplaceAll(result, "{"+k+"}", v)
	}

	// Replace dynamic tag placeholders
	// Find all app.tags.* placeholders and resolve them
	result = placeholderRegex.ReplaceAllStringFunc(result, func(match string) string {
		// Extract placeholder name (without braces)
		placeholder := match[1 : len(match)-1]

		if strings.HasPrefix(placeholder, tagPlaceholderPrefix) {
			tagKey := strings.TrimPrefix(placeholder, tagPlaceholderPrefix)

			return identity.Tags.Get(tagKey)
		}

		// Not a tag placeholder, leave unchanged (shouldn't happen after validation)
		return match
	})

	return result
}

// ValidateAndExpand combines validation and expansion in one call.
// It validates the pattern, checks required identity fields, and expands.
//
// Returns error if:
//   - Pattern contains unknown placeholders
//   - Pattern contains unclosed placeholders
//   - Pattern contains empty tag key ({app.tags.})
//   - Required tags are missing from identity
//   - App name is required but empty
func (t *NamingTemplate) ValidateAndExpand(identity AppIdentity) (string, error) {
	// Validate pattern syntax and placeholders
	if err := t.Validate(); err != nil {
		return "", err
	}

	// Check required tags
	requiredTags := t.RequiredTags()
	if len(requiredTags) > 0 {
		if err := identity.RequireTags(requiredTags...); err != nil {
			return "", err
		}
	}

	// Check app name if required
	if t.RequiresAppName() && strings.TrimSpace(identity.Name) == "" {
		return "", fmt.Errorf("naming pattern requires app.name but it is empty")
	}

	// Check app.env if required
	if t.RequiresEnv() && strings.TrimSpace(identity.Env) == "" {
		return "", fmt.Errorf("naming pattern requires app.env but it is empty")
	}

	return t.Expand(identity), nil
}
