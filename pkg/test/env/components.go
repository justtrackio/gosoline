package env

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/log"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

var componentFactories = map[string]componentFactory{}

type componentContainerDescription struct {
	containerConfig *containerConfig
	healthCheck     ComponentHealthCheck
}

type componentContainerDescriptions map[string]*componentContainerDescription

type componentFactory interface {
	Detect(config cfg.Config, manager *ComponentsConfigManager) error
	GetSettingsSchema() ComponentBaseSettingsAware
	DescribeContainers(settings interface{}) componentContainerDescriptions
	Component(config cfg.Config, logger log.Logger, container map[string]*container, settings interface{}) (Component, error)
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

func (c *baseComponent) failNow(failureMessage string, msgAndArgs ...interface{}) {
	assert.FailNow(c.t, failureMessage, msgAndArgs...)
}

type componentSkeleton struct {
	typ                   string
	name                  string
	settings              interface{}
	containerDescriptions componentContainerDescriptions
	containers            map[string]*container
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

		containerConfigs := factory.DescribeContainers(settings)

		skeleton := &componentSkeleton{
			typ:                   settings.GetType(),
			name:                  settings.GetName(),
			settings:              settings,
			containerDescriptions: containerConfigs,
			containers:            make(map[string]*container),
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
