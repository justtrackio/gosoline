package mdl

import (
	"fmt"
	"strings"
)

type ConfigProvider interface {
	GetString(key string, optionalDefault ...string) (string, error)
}

type ModelId struct {
	Project     string `cfg:"project" default:"{app_project}"`
	Environment string `cfg:"environment" default:"{env}"`
	Family      string `cfg:"family" default:"{app_family}"`
	Group       string `cfg:"group" default:"{app_group}"`
	Application string `cfg:"application" default:"{app_name}"`
	Name        string `cfg:"name"`
	Realm       string `cfg:"realm" default:"{app_project}-{env}-{app_family}-{app_group}"`
}

func (m *ModelId) String() string {
	return fmt.Sprintf("%s.%s.%s.%s", m.Project, m.Family, m.Group, m.Name)
}

func (m *ModelId) PadFromConfig(config ConfigProvider) error {
	var err error

	if m.Project == "" {
		if m.Project, err = config.GetString("app_project"); err != nil {
			return fmt.Errorf("could not get app_project: %w", err)
		}
	}

	if m.Environment == "" {
		if m.Environment, err = config.GetString("env"); err != nil {
			return fmt.Errorf("could not get env: %w", err)
		}
	}

	if m.Family == "" {
		if m.Family, err = config.GetString("app_family"); err != nil {
			return fmt.Errorf("could not get app_family: %w", err)
		}
	}

	if m.Group == "" {
		if m.Group, err = config.GetString("app_group"); err != nil {
			return fmt.Errorf("could not get app_group: %w", err)
		}
	}

	if m.Application == "" {
		if m.Application, err = config.GetString("app_name"); err != nil {
			return fmt.Errorf("could not get app_name: %w", err)
		}
	}

	if m.Realm == "" {
		if m.Realm, err = config.GetString("realm"); err != nil {
			return fmt.Errorf("could not get realm: %w", err)
		}
	}

	return nil
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
