package env

import (
	"fmt"
	"testing"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/stretchr/testify/assert"
)

var componentFactories = map[string]componentFactory{}

type ComponentContainerDescription struct {
	ContainerConfig  *ContainerConfig
	HealthCheck      ComponentHealthCheck
	ShutdownCallback ComponentShutdownCallback
}

type ComponentContainerDescriptions map[string]*ComponentContainerDescription

type componentFactory interface {
	Detect(config cfg.Config, manager *ComponentsConfigManager) error
	GetSettingsSchema() ComponentBaseSettingsAware
	DescribeContainers(settings any) ComponentContainerDescriptions
	Component(config cfg.Config, logger log.Logger, container map[string]*Container, settings any) (Component, error)
}

type (
	ComponentHealthCheck      func(container *Container) error
	ComponentShutdownCallback func(container *Container) func() error
)

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

type ContainerImageSettings struct {
	Auth       authSettings `cfg:"auth"`
	Repository string       `cfg:"repository"`
	Tag        string       `cfg:"tag"`
}

type ContainerBindingSettings struct {
	Host string `cfg:"host" default:"127.0.0.1"`
	Port int    `cfg:"port" default:"0"`
}

type ComponentContainerSettings struct {
	Image ContainerImageSettings `cfg:"image"`
	Tmpfs []TmpfsSettings        `cfg:"tmpfs"`
}

type TmpfsSettings struct {
	Path string `cfg:"path"`
	Size string `cfg:"size"`
	Mode string `cfg:"mode"`
}

type Component interface {
	SetT(t *testing.T)
}

type ComponentAddressAware interface {
	Address() string
}

type ComponentCfgOptionAware interface {
	CfgOptions() []cfg.Option
}

type baseComponent struct {
	t    *testing.T
	name string
}

func (c *baseComponent) SetT(t *testing.T) {
	c.t = t
}

func (c *baseComponent) failNow(failureMessage string, msgAndArgs ...any) {
	assert.FailNow(c.t, failureMessage, msgAndArgs...)
}

type componentSkeleton struct {
	typ                   string
	name                  string
	settings              ComponentBaseSettingsAware
	containerDescriptions ComponentContainerDescriptions
	containers            map[string]*Container
}

func (s componentSkeleton) id() string {
	return fmt.Sprintf("%s-%s", s.typ, s.name)
}

func buildComponentSkeletons(manager *ComponentsConfigManager) ([]*componentSkeleton, error) {
	var err error
	var allSettings []ComponentBaseSettingsAware
	skeletons := make([]*componentSkeleton, 0)

	if allSettings, err = manager.GetAllSettings(); err != nil {
		return nil, fmt.Errorf("can not read settings for components: %w", err)
	}

	for _, settings := range allSettings {
		factory, ok := componentFactories[settings.GetType()]

		if !ok {
			return nil, fmt.Errorf("there is no component of type %s available", settings.GetType())
		}

		containerConfigs := factory.DescribeContainers(settings)

		skeleton := &componentSkeleton{
			typ:                   settings.GetType(),
			name:                  settings.GetName(),
			settings:              settings,
			containerDescriptions: containerConfigs,
			containers:            make(map[string]*Container),
		}

		skeletons = append(skeletons, skeleton)
	}

	return skeletons, nil
}

func buildComponent(config cfg.Config, logger log.Logger, skeleton *componentSkeleton) (Component, error) {
	factory, ok := componentFactories[skeleton.typ]

	if !ok {
		return nil, fmt.Errorf("there is no component of type %s available", skeleton.typ)
	}

	return factory.Component(config, logger, skeleton.containers, skeleton.settings)
}
