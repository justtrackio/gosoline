package cfg

import (
	"errors"
	"fmt"
	"strings"

	"github.com/justtrackio/gosoline/pkg/funk"
)

// AppTags is a map of tag key-value pairs with helper methods.
type AppTags map[string]string

// Get returns the value for a tag key, or empty string if not present.
func (t AppTags) Get(key string) string {
	if t == nil {
		return ""
	}

	return t[key]
}

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

func (i AppIdentity) ToMap() map[string]string {
	mss := map[string]string{
		"app.name": i.Name,
		"app.env":  i.Env,
	}

	for key, value := range i.Tags {
		mss[fmt.Sprintf("app.tags.%s", key)] = value
	}

	return mss
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
	var err error
	var name, env string
	var tags map[string]string

	// app.name is required
	if name, err = config.GetString("app.name"); err != nil {
		return AppIdentity{}, fmt.Errorf("app.name: %w", err)
	}
	if strings.TrimSpace(name) == "" {
		return AppIdentity{}, errors.New("app.name: value is empty")
	}

	// app.env is required
	if env, err = config.GetString("app.env"); err != nil {
		return AppIdentity{}, fmt.Errorf("app.env: %w", err)
	}
	if strings.TrimSpace(env) == "" {
		return AppIdentity{}, errors.New("app.env: value is empty")
	}

	// Tags are optional
	if tags, err = config.GetStringMapString("app.tags", map[string]string{}); err != nil {
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
	var err error
	var tags map[string]string

	// Name and Env fields are needed from config
	if i.Name == "" {
		if i.Name, err = config.GetString("app.name"); err != nil {
			return fmt.Errorf("app.name: %w", err)
		}

		if strings.TrimSpace(i.Name) == "" {
			return errors.New("app.name: value is empty")
		}
	}

	if i.Env == "" {
		if i.Env, err = config.GetString("app.env"); err != nil {
			return fmt.Errorf("app.env: %w", err)
		}

		if strings.TrimSpace(i.Env) == "" {
			return errors.New("app.env: value is empty")
		}
	}

	// Merge tags: keep existing, add missing from config
	if tags, err = config.GetStringMapString("app.tags", map[string]string{}); err != nil {
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
