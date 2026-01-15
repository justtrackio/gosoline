package mdl

import (
	"errors"
	"fmt"
	"strings"
)

// ConfigProvider is an interface for reading config values.
// It is implemented by cfg.Config.
type ConfigProvider interface {
	GetString(key string, optionalDefault ...string) (string, error)
}

// ModelId represents the identity of a model, extending the application
// identity with a model name.
//
// The struct tag defaults use the macro placeholders:
//   - {app.name} for application name
//   - {app.tags.project} for project
//   - {app.tags.family} for family
//   - {app.tags.group} for group
//   - {app.env} for environment
type ModelId struct {
	Project     string `cfg:"project" default:"{app.tags.project}"`
	Environment string `cfg:"environment" default:"{app.env}"`
	Family      string `cfg:"family" default:"{app.tags.family}"`
	Group       string `cfg:"group" default:"{app.tags.group}"`
	Application string `cfg:"application" default:"{app.name}"`
	Name        string `cfg:"name"`
}

func (m *ModelId) String() string {
	return fmt.Sprintf("%s.%s.%s.%s", m.Project, m.Family, m.Group, m.Name)
}

// PadFromConfig fills in empty fields of ModelId from config.
//
// This method requires the following config keys if the corresponding
// ModelId fields are empty:
//   - app.tags.project
//   - app.tags.family
//   - app.tags.group
//   - app.name
//   - app.env
//
// This method is useful when you have a partially populated ModelId
// (e.g., from struct tag defaults) and want to fill remaining fields.
func (m *ModelId) PadFromConfig(config ConfigProvider) error {
	var errs []error

	if m.Project == "" {
		m.Project, errs = padStringFieldFromConfig(config, "app.tags.project", errs)
	}

	if m.Environment == "" {
		m.Environment, errs = padStringFieldFromConfig(config, "app.env", errs)
	}

	if m.Family == "" {
		m.Family, errs = padStringFieldFromConfig(config, "app.tags.family", errs)
	}

	if m.Group == "" {
		m.Group, errs = padStringFieldFromConfig(config, "app.tags.group", errs)
	}

	if m.Application == "" {
		m.Application, errs = padStringFieldFromConfig(config, "app.name", errs)
	}

	if len(errs) > 0 {
		return fmt.Errorf("could not pad ModelId from config: %w", errors.Join(errs...))
	}

	return nil
}

// padStringFieldFromConfig reads a required string field from config and appends errors if needed.
func padStringFieldFromConfig(config ConfigProvider, key string, errs []error) (string, []error) {
	value, err := config.GetString(key)
	switch {
	case err != nil:
		errs = append(errs, fmt.Errorf("%s: %w", key, err))
	case value == "":
		errs = append(errs, fmt.Errorf("%s: value is empty", key))
	}

	return value, errs
}

func ModelIdFromString(str string) (ModelId, error) {
	parts := strings.Split(str, ".")

	if len(parts) != 4 {
		return ModelId{}, fmt.Errorf("the model id (%s) should consist out of 4 parts", str)
	}

	modelId := ModelId{
		Project: parts[0],
		Family:  parts[1],
		Group:   parts[2],
		Name:    parts[3],
	}

	return modelId, nil
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
