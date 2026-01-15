package cfg

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/justtrackio/gosoline/pkg/funk"
)

// AppIdentity represents the resolved application identity.
// It is used throughout gosoline for resource naming and identification.
//
// Unlike the old AppId type, AppIdentity uses dynamic tags rather than
// fixed fields for project/family/group. This allows arbitrary tags
// while still supporting the common naming patterns.
//
// Configuration structure:
//
//	app:
//	  env: production    # required globally
//	  name: myapp        # required
//	  tags:              # optional globally, but some subsystems require specific tags
//	    project: ...
//	    family: ...
//	    group: ...
//	    custom: ...      # any additional tags
type AppIdentity struct {
	Name string  `json:"name"`
	Env  string  `json:"env"`
	Tags AppTags `json:"tags"`
}

// GetAppIdentityFromConfig reads the application identity from config.
//
// This function requires:
//   - "app.name" to be present and non-empty
//   - "app.env" to be present and non-empty
//
// Tags are optional at this level. Subsystems that require specific tags
// (e.g., project, family, group for naming) should call RequireTags().
func GetAppIdentityFromConfig(config Config) (AppIdentity, error) {
	// app.name is required
	name, err := config.GetString("app.name")
	if err != nil {
		return AppIdentity{}, fmt.Errorf("app.name: %w", err)
	}

	if strings.TrimSpace(name) == "" {
		return AppIdentity{}, errors.New("app.name: value is empty")
	}

	// app.env is required
	env, err := config.GetString("app.env")
	if err != nil {
		return AppIdentity{}, fmt.Errorf("app.env: %w", err)
	}

	if strings.TrimSpace(env) == "" {
		return AppIdentity{}, errors.New("app.env: value is empty")
	}

	// Tags are optional
	tags, err := config.GetStringMapString("app.tags", map[string]string{})
	if err != nil {
		return AppIdentity{}, fmt.Errorf("app.tags: %w", err)
	}

	return AppIdentity{
		Name: name,
		Env:  env,
		Tags: tags,
	}, nil
}

// PadFromConfig fills in empty fields of AppIdentity from config.
//
// Behavior:
//   - If Name is empty, fills from app.name
//   - If Env is empty, fills from app.env (required, will error if missing/empty)
//   - If Tags is nil or empty, fills from app.tags
//   - Existing tag keys are NOT overwritten; only missing keys are added
//
// This method is useful when you have a partially populated AppIdentity
// (e.g., from struct tag defaults) and want to fill remaining fields.
func (i *AppIdentity) PadFromConfig(config Config) error {
	// Name and Env fields are needed from config
	needsName := i.Name == ""
	needsEnv := i.Env == ""

	if needsName {
		name, err := config.GetString("app.name")
		if err != nil {
			return fmt.Errorf("app.name: %w", err)
		}

		if strings.TrimSpace(name) == "" {
			return errors.New("app.name: value is empty")
		}

		i.Name = name
	}

	if needsEnv {
		env, err := config.GetString("app.env")
		if err != nil {
			return fmt.Errorf("app.env: %w", err)
		}

		if strings.TrimSpace(env) == "" {
			return errors.New("app.env: value is empty")
		}

		i.Env = env
	}

	// Merge tags: keep existing, add missing from config
	tags, err := config.GetStringMapString("app.tags", map[string]string{})
	if err != nil {
		return fmt.Errorf("app.tags: %w", err)
	}

	if i.Tags == nil {
		i.Tags = make(AppTags)
	}

	for key, value := range tags {
		if _, exists := i.Tags[key]; !exists {
			i.Tags[key] = value
		}
	}

	return nil
}

// RequireTags validates that the specified tag keys are present and non-empty.
// Whitespace-only values are treated as missing.
//
// This method should be called by subsystems that require specific tags
// for naming or identification purposes. For example, kafka topic naming
// might call identity.RequireTags("project", "family", "group").
//
// Returns an error listing all missing tags in sorted order, e.g.:
//
//	"missing required tags: family, project"
func (i *AppIdentity) RequireTags(keys ...string) error {
	var missing []string

	for _, key := range keys {
		value := strings.TrimSpace(i.Tags.Get(key))
		if value == "" {
			missing = append(missing, key)
		}
	}

	if len(missing) > 0 {
		slices.Sort(missing)

		return fmt.Errorf("missing required tags: %s", strings.Join(missing, ", "))
	}

	return nil
}

// String returns a canonical string representation of the identity.
// It joins non-empty components in the order: project, env, family, group, name.
// Empty components are skipped.
//
// Examples:
//   - Full: "myproject-production-myfamily-mygroup-myapp"
//   - Partial: "production-myapp" (if only env and name are set)
func (i *AppIdentity) String() string {
	elements := []string{
		i.Tags.Get("project"),
		i.Env,
		i.Tags.Get("family"),
		i.Tags.Get("group"),
		i.Name,
	}
	elements = funk.Filter(elements, func(element string) bool {
		return element != ""
	})

	return strings.Join(elements, "-")
}
