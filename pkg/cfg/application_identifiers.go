package cfg

import (
	"errors"
	"fmt"
	"strings"

	"github.com/justtrackio/gosoline/pkg/funk"
)

// AppTags is a map of tag key-value pairs with helper methods.
type AppTags map[string]string

// AppIdentity represents the resolved application identity.
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
type AppIdentity struct {
	Name string  `json:"name" cfg:"name"`
	Env  string  `json:"env" cfg:"env"`
	Tags AppTags `json:"tags" cfg:"tags"`
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

// GetAppIdentity reads the application identity from config.
//
// This function requires:
//   - "app.name" to be present and non-empty
//   - "app.env" to be present and non-empty
func GetAppIdentity(config Config) (AppIdentity, error) {
	identity := AppIdentity{}

	if err := config.UnmarshalKey("app", &identity); err != nil {
		return AppIdentity{}, fmt.Errorf("failed to unmarshal app identity: %w", err)
	}

	return identity, nil
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
		i.Tags["project"],
		i.Env,
		i.Tags["family"],
		i.Tags["group"],
		i.Name,
	}
	elements = funk.Filter(elements, func(element string) bool {
		return element != ""
	})

	return strings.Join(elements, "-")
}
