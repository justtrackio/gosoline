package env

import (
	"context"
	_ "embed"
	"fmt"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/encoding/yaml"
	"github.com/justtrackio/gosoline/pkg/fixtures"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/stretchr/testify/assert"
)

//go:embed config.default.yml
var configDefault []byte

type Environment struct {
	componentOptions []ComponentOption
	configOptions    []ConfigOption
	loggerOptions    []LoggerOption

	t          *testing.T
	ctx        context.Context
	config     cfg.GosoConf
	logger     RecordingLogger
	filesystem *filesystem
	runner     *containerRunner
	components *ComponentsContainer
}

func NewEnvironment(t *testing.T, options ...Option) (*Environment, error) {
	var err error

	env := &Environment{
		t: t,
	}

	if err = env.init(options...); err != nil {
		return nil, fmt.Errorf("can not initialize environment: %w", err)
	}

	var skeletons []*componentSkeleton
	var component Component
	components := NewComponentsContainer()
	componentConfigManager := NewComponentsConfigManager(env.config)

	for _, opt := range env.componentOptions {
		if err := opt(componentConfigManager); err != nil {
			return nil, fmt.Errorf("can apply component option: %w", err)
		}
	}

	for typ, factory := range componentFactories {
		if err = factory.Detect(env.config, componentConfigManager); err != nil {
			return env, fmt.Errorf("can not autodetect components for %s: %w", typ, err)
		}
	}

	if skeletons, err = buildComponentSkeletons(componentConfigManager); err != nil {
		return env, fmt.Errorf("can not create component skeletons: %w", err)
	}

	if env.runner, err = NewContainerRunner(env.config, env.logger); err != nil {
		return env, fmt.Errorf("can not create container runner: %w", err)
	}

	if err := env.runner.RunContainers(skeletons); err != nil {
		return env, err
	}

	for _, skeleton := range skeletons {
		if component, err = buildComponent(env.config, env.logger, skeleton); err != nil {
			return env, fmt.Errorf("can not build component %s: %w", skeleton.id(), err)
		}

		component.SetT(t)
		components.Add(skeleton.typ, skeleton.name, component)
	}

	if err = env.config.Option(components.GetCfgOptions()...); err != nil {
		return nil, fmt.Errorf("can not apply cfg options from components: %w", err)
	}

	env.components = components

	return env, nil
}

func (e *Environment) init(options ...Option) error {
	start := time.Now()

	var err error
	var logger RecordingLogger
	var cfgPostProcessors map[string]int

	defaults := make(map[string]any)
	if err = yaml.Unmarshal(configDefault, &defaults); err != nil {
		return fmt.Errorf("can not read default configurion: %w", err)
	}
	options = append([]Option{WithConfigMap(defaults)}, options...)

	for _, opt := range options {
		opt(e)
	}

	config := cfg.New()
	for _, opt := range e.configOptions {
		if err := opt(config); err != nil {
			return fmt.Errorf("can apply config option: %w", err)
		}
	}

	if cfgPostProcessors, err = cfg.ApplyPostProcessors(config); err != nil {
		return fmt.Errorf("can not apply post processor on config: %w", err)
	}

	if logger, err = NewRecordingConsoleLogger(e.loggerOptions...); err != nil {
		return fmt.Errorf("can apply logger option: %w", err)
	}

	defer func() {
		logger.Debug("booted env in %s", time.Since(start))
	}()

	for name, priority := range cfgPostProcessors {
		logger.Info("applied priority %d config post processor '%s'", priority, name)
	}

	e.ctx = appctx.WithContainer(context.Background())
	e.logger = logger
	e.config = config
	e.filesystem = newFilesystem(e.t)

	return nil
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

func (e *Environment) Logs() LogRecords {
	return e.logger.Records()
}

func (e *Environment) ResetLogs() {
	e.logger.Reset()
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

func (e *Environment) LoadFixtureSet(factory fixtures.FixtureSetsFactory, postProcessorFactories ...fixtures.PostProcessorFactory) error {
	return e.LoadFixtureSets([]fixtures.FixtureSetsFactory{factory}, postProcessorFactories...)
}

func (e *Environment) LoadFixtureSets(factories []fixtures.FixtureSetsFactory, postProcessorFactories ...fixtures.PostProcessorFactory) error {
	if len(factories) == 0 {
		return nil
	}

	var err error
	var loader fixtures.FixtureLoader
	var fixtureSets, allFixtureSets []fixtures.FixtureSet

	for _, factory := range factories {
		if fixtureSets, err = factory(e.ctx, e.config, e.logger, "default"); err != nil {
			return fmt.Errorf("failed to create fixture set: %w", err)
		}

		allFixtureSets = append(allFixtureSets, fixtureSets...)
	}

	if loader, err = fixtures.NewFixtureLoader(e.ctx, e.config, e.logger, postProcessorFactories...); err != nil {
		return fmt.Errorf("failed to create fixture loader: %w", err)
	}

	if err = loader.Load(e.ctx, "default", allFixtureSets); err != nil {
		return fmt.Errorf("failed to load all fixture sets: %w", err)
	}

	return nil
}
