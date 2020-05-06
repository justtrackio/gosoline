package env

import (
	"fmt"
	"github.com/applike/gosoline/pkg/application"
)

type ComponentsContainer struct {
	components map[string]map[string]Component
}

func NewComponentsContainer() *ComponentsContainer {
	return &ComponentsContainer{
		components: make(map[string]map[string]Component),
	}
}

func (c *ComponentsContainer) Add(typ string, name string, component Component) {
	if _, ok := c.components[typ]; !ok {
		c.components[typ] = make(map[string]Component)
	}

	c.components[typ][name] = component
}

func (c *ComponentsContainer) Get(typ string, name string) (Component, error) {
	if _, ok := c.components[typ]; !ok {
		return nil, fmt.Errorf("there is no component with name %s of type %s", name, typ)
	}

	if _, ok := c.components[typ][name]; !ok {
		return nil, fmt.Errorf("there is no component with name %s of type %s", name, typ)
	}

	return c.components[typ][name], nil
}

func (c *ComponentsContainer) GetAll() []Component {
	all := make([]Component, 0)

	for _, components := range c.components {
		for _, component := range components {
			all = append(all, component)
		}
	}

	return all
}

func (c *ComponentsContainer) GetApplicationOptions() []application.Option {
	var ok bool
	var appOptionAware ComponentAppOptionAware
	var appOptions = make([]application.Option, 0)

	for _, component := range c.GetAll() {
		if appOptionAware, ok = component.(ComponentAppOptionAware); !ok {
			continue
		}

		appOptions = append(appOptions, appOptionAware.AppOptions()...)
	}

	return appOptions
}
