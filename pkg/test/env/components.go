package env

import (
	"fmt"
	"github.com/applike/gosoline/pkg/application"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

var componentFactories = map[string]componentFactory{}

type componentFactory interface {
	Detect(config cfg.Config, manager *ComponentsConfigManager) error
	GetSettingsSchema() ComponentBaseSettingsAware
	ConfigureContainer(settings interface{}) *containerConfig
	HealthCheck(settings interface{}) ComponentHealthCheck
	Component(config cfg.Config, logger mon.Logger, container *container, settings interface{}) (Component, error)
}

type ComponentHealthCheck func(container *container) error

type ComponentBaseSettingsAware interface {
	GetName() string
	GetType() string
	SetName(name string)
	SetType(typ string)
}

type ComponentBaseSettings struct {
	Name string `cfg:"name" default:"default"`
	Type string `cfg:"type" validate:"required"`
}

func (c *ComponentBaseSettings) GetName() string {
	return c.Name
}

func (c *ComponentBaseSettings) GetType() string {
	return c.Type
}

func (c *ComponentBaseSettings) SetName(name string) {
	c.Name = name
}

func (c *ComponentBaseSettings) SetType(typ string) {
	c.Type = typ
}

type ComponentContainerSettings struct {
	ExpireAfter time.Duration   `cfg:"expire_after" default:"60s"`
	Tmpfs       []TmpfsSettings `cfg:"tmpfs"`
}

type TmpfsSettings struct {
	Path string `cfg:"path"`
	Size string `cfg:"size"`
	Mode string `cfg:"mode"`
}

type Component interface {
	SetT(t *testing.T)
}

type ComponentAppOptionAware interface {
	AppOptions() []application.Option
}

type baseComponent struct {
	t    *testing.T
	name string
}

func (c *baseComponent) SetT(t *testing.T) {
	c.t = t
}

func (c *baseComponent) failNow(failureMessage string, msgAndArgs ...interface{}) {
	assert.FailNow(c.t, failureMessage, msgAndArgs...)
}

type componentSkeleton struct {
	typ             string
	name            string
	settings        interface{}
	containerConfig *containerConfig
	healthCheck     func(container *container) error
}

func (s componentSkeleton) id() string {
	return fmt.Sprintf("%s-%s", s.typ, s.name)
}

func buildComponentSkeletons(manager *ComponentsConfigManager) ([]*componentSkeleton, error) {
	var err error
	var allSettings []ComponentBaseSettingsAware
	var skeletons = make([]*componentSkeleton, 0)

	if allSettings, err = manager.GetAllSettings(); err != nil {
		return nil, fmt.Errorf("can not read settings for components: %w", err)
	}

	for _, settings := range allSettings {
		factory, ok := componentFactories[settings.GetType()]

		if !ok {
			return nil, fmt.Errorf("there is no component of type %s available", settings.GetType())
		}

		containerConfig := factory.ConfigureContainer(settings)
		healthCheck := factory.HealthCheck(settings)

		skeleton := &componentSkeleton{
			typ:             settings.GetType(),
			name:            settings.GetName(),
			settings:        settings,
			containerConfig: containerConfig,
			healthCheck:     healthCheck,
		}

		skeletons = append(skeletons, skeleton)
	}

	return skeletons, nil
}

func buildComponent(config cfg.Config, logger mon.Logger, skeleton *componentSkeleton, container *container) (Component, error) {
	factory, ok := componentFactories[skeleton.typ]

	if !ok {
		return nil, fmt.Errorf("there is no component of type %s available", container.typ)
	}

	return factory.Component(config, logger, container, skeleton.settings)
}
