package env

import (
	"fmt"
	"github.com/applike/gosoline/pkg/application"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/stretchr/testify/assert"
	"testing"
)

type Environment struct {
	configOptions []ConfigOption
	loggerOptions []LoggerOption

	t          *testing.T
	runner     *containerRunner
	components *ComponentsContainer
}

func NewEnvironment(t *testing.T, options ...Option) (*Environment, error) {
	env := &Environment{
		t: t,
	}

	for _, opt := range options {
		opt(env)
	}

	config := cfg.New()
	for _, opt := range env.configOptions {
		if err := opt(config); err != nil {
			return nil, fmt.Errorf("can apply config option: %w", err)
		}
	}

	logger := mon.NewLogger()
	for _, opt := range env.loggerOptions {
		if err := opt(config, logger); err != nil {
			return nil, fmt.Errorf("can apply logger option: %w", err)
		}
	}

	env.runner = NewContainerRunner(config, logger)

	var err error
	var skeletons map[string]componentSkeleton
	var component Component
	var containers = make(map[string]*container)
	var components = NewComponentsContainer()
	var componentConfigManger = NewComponentsConfigManager(config)

	for typ, factory := range componentFactories {
		if err = factory.Detect(config, componentConfigManger); err != nil {
			return env, fmt.Errorf("can not autodetect components for %s: %w", typ, err)
		}
	}

	if skeletons, err = buildComponentSkeletons(componentConfigManger); err != nil {
		return env, fmt.Errorf("can not create component skeletons: %w", err)
	}

	if containers, err = env.runner.RunContainers(skeletons); err != nil {
		return env, err
	}

	for name, skeleton := range skeletons {
		if component, err = buildComponent(skeleton, containers[name]); err != nil {
			return env, fmt.Errorf("can not build component %s: %w", name, err)
		}

		component.SetT(t)
		components.Add(skeleton.typ, skeleton.name, component)
	}

	env.components = components

	return env, nil
}

func (e *Environment) addConfigOption(opt ConfigOption) {
	e.configOptions = append(e.configOptions, opt)
}

func (e *Environment) addLoggerOption(opt LoggerOption) {
	e.loggerOptions = append(e.loggerOptions, opt)
}

func (e *Environment) Stop() error {
	return e.runner.Stop()
}

func (e *Environment) ApplicationOptions() []application.Option {
	return e.components.GetApplicationOptions()
}

func (e *Environment) Component(typ string, name string) Component {
	var err error
	var component Component

	if component, err = e.components.Get(typ, name); err != nil {
		assert.FailNow(e.t, "can not get component", err.Error())
	}

	return component
}

func (e *Environment) MySql(name string) *mysqlComponent {
	return e.Component(componentMySql, name).(*mysqlComponent)
}

func (e *Environment) StreamInput(name string) *streamInputComponent {
	return e.Component(componentStreamInput, name).(*streamInputComponent)
}
