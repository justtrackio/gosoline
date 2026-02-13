package cfg

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/spf13/cast"
)

var (
	patternRegex   = regexp.MustCompile(`(?m)\{([^\}]+)\}`)
	namespaceRegex = regexp.MustCompile(`\{[^}]+\}|[^.]+`)
)

// Tags is a map of tag key-value pairs with helper methods.
type Tags map[string]string

// Identity represents the resolved application identity.
// It is used throughout gosoline for resource naming and identification.
//
// Configuration structure:
//
//	app:
//	  env: production    # required
//	  name: myapp        # required
//	  tags:              # optional
//	    project: ...
//	    family: ...
//	    group: ...
//	    custom: ...      # any additional tags
type Identity struct {
	Env            string `cfg:"env" json:"env" yaml:"env"`
	Name           string `cfg:"name" json:"name" yaml:"name"`
	Tags           Tags   `cfg:"tags" json:"tags" yaml:"tags"`
	Namespace      string `cfg:"namespace,nodecode" json:"-" yaml:"-"`
	namespaceParts []string
}

func (i Identity) Format(pattern string, delimiter string, args ...map[string]string) (string, error) {
	var err error
	var values map[string]string
	var result string

	if values, err = i.ToPlaceholders(delimiter, args...); err != nil {
		return "", fmt.Errorf("failed to get placeholders: %w", err)
	}

	if result, err = i.format(pattern, values); err != nil {
		return "", err
	}

	result = strings.TrimSpace(result)

	if result == "" {
		return "", fmt.Errorf("formatted result is empty for pattern %q", pattern)
	}

	return result, nil
}

func (i Identity) FormatNamespace(delimiter string, args ...map[string]string) (string, error) {
	var err error
	var values map[string]string

	if values, err = i.ToPlaceholders(delimiter, args...); err != nil {
		return "", fmt.Errorf("failed to get placeholders: %w", err)
	}

	return values["app.namespace"], nil
}

func (i Identity) format(pattern string, args map[string]string) (string, error) {
	result := pattern
	matches := patternRegex.FindAllStringSubmatch(pattern, -1)

	for _, match := range matches {
		key := match[1]
		value, ok := args[key]

		if !ok {
			return "", fmt.Errorf("unknown placeholder {%s} in pattern %q", key, pattern)
		}

		if value == "" {
			return "", fmt.Errorf("placeholder {%s} resolved to an empty value in pattern %q", key, pattern)
		}

		result = strings.ReplaceAll(result, match[0], value)
	}

	return result, nil
}

func (i Identity) ToPlaceholders(delimiter string, args ...map[string]string) (map[string]string, error) {
	var err error

	values := map[string]string{
		"app.name": i.Name,
		"app.env":  i.Env,
	}

	for key, value := range i.Tags {
		values[fmt.Sprintf("app.tags.%s", key)] = value
	}

	for _, a := range args {
		values = funk.MergeMaps(values, a)
	}

	namespacePattern := strings.Join(i.namespaceParts, delimiter)
	if values["app.namespace"], err = i.format(namespacePattern, values); err != nil {
		return nil, fmt.Errorf("failed to format app.namespace: %w", err)
	}

	return values, nil
}

// GetAppIdentity reads the application identity from config.
//
// This function requires:
//   - "app.name" to be present and non-empty
//   - "app.env" to be present and non-empty
func GetAppIdentity(config Config) (Identity, error) {
	identity := &Identity{}

	if err := identity.PadFromConfig(config); err != nil {
		return Identity{}, fmt.Errorf("failed to pad app identity from config: %w", err)
	}

	return *identity, nil
}

// PadFromConfig fills in empty fields of Identity from config.
//
// Behavior:
//   - If Name is empty, fills from app.name
//   - If Env is empty, fills from app.env (required, will error if missing/empty)
//   - If Tags is nil or empty, fills from app.tags
//   - Existing tag keys are NOT overwritten; only missing keys are added
//
// This method is useful when you have a partially populated Identity
// (e.g., from struct tag defaults) and want to fill remaining fields.
func (i *Identity) PadFromConfig(config Config) error {
	var err error
	var tags map[string]string
	var namespace any

	// Name and Env fields are needed from config
	if i.Name == "" {
		if i.Name, err = config.GetString("app.name"); err != nil {
			return fmt.Errorf("app.name: %w", err)
		}

		i.Name = strings.TrimSpace(i.Name)

		if i.Name == "" {
			return errors.New("app.name: value is empty")
		}
	}

	if i.Env == "" {
		if i.Env, err = config.GetString("app.env"); err != nil {
			return fmt.Errorf("app.env: %w", err)
		}

		i.Env = strings.TrimSpace(i.Env)

		if i.Env == "" {
			return errors.New("app.env: value is empty")
		}
	}

	// Merge tags: keep existing, add missing from config
	if tags, err = config.GetStringMapString("app.tags", map[string]string{}); err != nil {
		return fmt.Errorf("app.tags: %w", err)
	}

	i.Tags = funk.MergeMaps(tags, i.Tags)

	if i.Namespace == "" {
		if namespace, err = config.Get("app.namespace", ""); err != nil {
			return fmt.Errorf("can not get app.namespace from config: %w", err)
		}

		if i.Namespace, err = cast.ToStringE(namespace); err != nil {
			return fmt.Errorf("app.namespace %q is not a string: %w", namespace, err)
		}
	}

	i.namespaceParts = namespaceRegex.FindAllString(i.Namespace, -1)

	return nil
}
