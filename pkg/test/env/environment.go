package env

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/fixtures"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/stretchr/testify/assert"
)

type Environment struct {
	componentOptions []ComponentOption
	configOptions    []ConfigOption
	loggerOptions    []LoggerOption

	t             *testing.T
	ctx           context.Context
	config        cfg.GosoConf
	logger        log.GosoLogger
	filesystem    *filesystem
	fixtureLoader fixtures.FixtureLoader
	runner        *containerRunner
	components    *ComponentsContainer
}

func NewEnvironment(t *testing.T, options ...Option) (*Environment, error) {
	start := time.Now()

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

	var err error
	var logger log.GosoLogger
	var cfgPostProcessors map[string]int

	if cfgPostProcessors, err = cfg.ApplyPostProcessors(config); err != nil {
		return nil, fmt.Errorf("can not apply post processor on config: %w", err)
	}

	if logger, err = NewConsoleLogger(env.loggerOptions...); err != nil {
		return nil, fmt.Errorf("can apply logger option: %w", err)
	}

	defer func() {
		logger.Debug("booted env in %s", time.Since(start))
	}()

	for name, priority := range cfgPostProcessors {
		logger.Info("applied priority %d config post processor '%s'", priority, name)
	}

	var skeletons []*componentSkeleton
	var component Component
	components := NewComponentsContainer()
	componentConfigManager := NewComponentsConfigManager(config)

	for _, opt := range env.componentOptions {
		if err := opt(componentConfigManager); err != nil {
			return nil, fmt.Errorf("can apply component option: %w", err)
		}
	}

	env.ctx = appctx.WithContainer(context.Background())
	env.config = config
	env.logger = logger
	env.filesystem = newFilesystem(t)
	env.fixtureLoader = fixtures.NewFixtureLoader(env.ctx, env.config, env.logger)

	for typ, factory := range componentFactories {
		if err = factory.Detect(config, componentConfigManager); err != nil {
			return env, fmt.Errorf("can not autodetect components for %s: %w", typ, err)
		}
	}

	if skeletons, err = buildComponentSkeletons(componentConfigManager); err != nil {
		return env, fmt.Errorf("can not create component skeletons: %w", err)
	}

	if env.runner, err = NewContainerRunner(config, logger); err != nil {
		return env, fmt.Errorf("can not create container runner: %w", err)
	}

	if err = env.runner.RunContainers(skeletons); err != nil {
		return env, err
	}

	for _, skeleton := range skeletons {
		if component, err = buildComponent(config, logger, skeleton); err != nil {
			return env, fmt.Errorf("can not build component %s: %w", skeleton.id(), err)
		}

		component.SetT(t)
		components.Add(skeleton.typ, skeleton.name, component)
	}

	if err = config.Option(components.GetCfgOptions()...); err != nil {
		return nil, fmt.Errorf("can not apply cfg options from components: %w", err)
	}

	env.components = components

	return env, nil
}

func (e *Environment) addComponentOption(opt ComponentOption) {
	e.componentOptions = append(e.componentOptions, opt)
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

func (e *Environment) Context() context.Context {
	return e.ctx
}

func (e *Environment) Config() cfg.GosoConf {
	return e.config
}

func (e *Environment) Logger() log.GosoLogger {
	return e.logger
}

func (e *Environment) Clock() clock.Clock {
	return clock.Provider
}

func (e *Environment) Filesystem() *filesystem {
	return e.filesystem
}

func (e *Environment) Component(typ string, name string) Component {
	var err error
	var component Component

	if component, err = e.components.Get(typ, name); err != nil {
		assert.FailNow(e.t, "can not get component", err.Error())
	}

	return component
}

func (e *Environment) Redis(name string) *RedisComponent {
	return e.Component(componentRedis, name).(*RedisComponent)
}

func (e *Environment) S3(name string) *S3Component {
	return e.Component(componentS3, name).(*S3Component)
}

func (e *Environment) DynamoDb(name string) *DdbComponent {
	return e.Component(componentDdb, name).(*DdbComponent)
}

func (e *Environment) Localstack(name string) *localstackComponent {
	return e.Component(ComponentLocalstack, name).(*localstackComponent)
}

func (e *Environment) MySql(name string) *mysqlComponent {
	return e.Component(componentMySql, name).(*mysqlComponent)
}

func (e *Environment) Wiremock(name string) *wiremockComponent {
	return e.Component(componentWiremock, name).(*wiremockComponent)
}

func (e *Environment) StreamInput(name string) *StreamInputComponent {
	return e.Component(componentStreamInput, name).(*StreamInputComponent)
}

func (e *Environment) StreamOutput(name string) *streamOutputComponent {
	return e.Component(componentStreamOutput, name).(*streamOutputComponent)
}

func (e *Environment) LoadFixtureBuilderFactories(factories ...fixtures.FixtureBuilderFactory) error {
	if len(factories) == 0 {
		return nil
	}

	for _, factory := range factories {
		var err error
		var fixtureBuilder fixtures.FixtureBuilder

		if fixtureBuilder, err = factory(e.ctx); err != nil {
			return fmt.Errorf("can not build fixture builder: %w", err)
		}

		if err = e.fixtureLoader.Load(e.ctx, fixtureBuilder.Fixtures()); err != nil {
			return err
		}
	}

	return nil
}
