package env

import (
	"fmt"
	"github.com/applike/gosoline/pkg/application"
	"github.com/applike/gosoline/pkg/cfg"
	"testing"
	"time"
)

var componentFactories = map[string]componentFactory{
	componentCloudwatch:  new(cloudwatchFactory),
	componentMySql:       new(mysqlFactory),
	componentStreamInput: new(streamInputFactory),
}

type componentFactory interface {
	Detect(config cfg.Config, manager *ComponentsConfigManager) error
	GetSettingsSchema() ComponentBaseSettingsAware
	ConfigureContainer(settings interface{}) *containerConfig
	HealthCheck(settings interface{}) ComponentHealthCheck
	Component(settings interface{}, container *container) (Component, error)
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
	ExpireAfter time.Duration `cfg:"expire_after" default:"60s"`
}

type Component interface {
	SetT(t *testing.T)
}

type ComponentAppOptionAware interface {
	AppOptions() []application.Option
}

type baseComponent struct {
	t *testing.T
}

func (c *baseComponent) SetT(t *testing.T) {
	c.t = t
}

type componentSkeleton struct {
	typ             string
	name            string
	settings        interface{}
	containerConfig *containerConfig
	healthCheck     func(container *container) error
}

func buildComponentSkeletons(manager *ComponentsConfigManager) (map[string]componentSkeleton, error) {
	var err error
	var allSettings []ComponentBaseSettingsAware
	var skeletons = make(map[string]componentSkeleton)

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

		skeletons[settings.GetName()] = componentSkeleton{
			typ:             settings.GetType(),
			name:            settings.GetName(),
			settings:        settings,
			containerConfig: containerConfig,
			healthCheck:     healthCheck,
		}
	}

	return skeletons, nil
}

func buildComponent(skeleton componentSkeleton, container *container) (Component, error) {
	factory, ok := componentFactories[skeleton.typ]

	if !ok {
		return nil, fmt.Errorf("there is no component of type %s available", skeleton.typ)
	}

	return factory.Component(skeleton.settings, container)
}
