package kernel

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/coffin"
	"github.com/justtrackio/gosoline/pkg/log"
)

type factory struct {
	ctx       context.Context
	config    cfg.Config
	logger    log.Logger
	blueprint *blueprint

	kernel      *kernel
	middlewares []Middleware
	stages      stages
}

func NewFactory(ctx context.Context, config cfg.Config, logger log.Logger, bp *blueprint) (*factory, error) {
	var err error

	factory := &factory{
		ctx:       ctx,
		config:    config,
		logger:    logger.WithChannel("kernel"),
		blueprint: bp,

		middlewares: make([]Middleware, 0),
		stages:      make(stages),
	}

	if factory.kernel, err = newKernel(ctx, config, logger); err != nil {
		return nil, fmt.Errorf("can not create kernel: %w", err)
	}

	if err = factory.build(); err != nil {
		return nil, fmt.Errorf("can not build kernel factory: %w", err)
	}

	return factory, nil
}

func (f *factory) GetKernel() (Kernel, error) {
	f.kernel.init(f.middlewares, f.stages)

	for _, opt := range f.blueprint.kernelOptions {
		opt(f.kernel)
	}

	return f.kernel, nil
}

func (f *factory) build() (err error) {
	defer func() {
		if err != nil {
			return
		}

		err = coffin.ResolveRecovery(recover())
	}()

	for _, mf := range f.blueprint.middlewareFactories {
		if err := f.buildMiddleware(mf.factory, mf.position); err != nil {
			return err
		}
	}

	for _, mf := range f.blueprint.moduleFactories {
		if err := f.buildModuleFactory(mf.name, mf.factory, mf.options...); err != nil {
			return err
		}
	}

	for _, mf := range f.blueprint.multiModuleFactories {
		if err := f.buildMultiModuleFactory(mf); err != nil {
			return err
		}
	}

	if !f.stages.hasModules() {
		return fmt.Errorf("no modules to run")
	}

	if f.stages.countForegroundModules() == 0 {
		return fmt.Errorf("no foreground modules to run")
	}

	return nil
}

func (f *factory) buildModuleFactory(name string, factory ModuleFactory, opts ...ModuleOption) error {
	var err error
	var module Module

	if module, err = factory(f.ctx, f.config, f.logger); err != nil {
		return fmt.Errorf("can not build module %s: %w", name, err)
	}

	if err = f.addModuleToStage(name, module, opts); err != nil {
		return fmt.Errorf("can not add module to stage: %w", err)
	}

	return nil
}

func (f *factory) buildMultiModuleFactory(factory ModuleMultiFactory) error {
	var err error
	var moduleFactories map[string]ModuleFactory

	if moduleFactories, err = factory(f.ctx, f.config, f.logger); err != nil {
		return err
	}

	for name, moduleFactory := range moduleFactories {
		if err := f.buildModuleFactory(name, moduleFactory); err != nil {
			return err
		}
	}

	return nil
}

func (f *factory) buildMiddleware(middlewareFactory MiddlewareFactory, position Position) error {
	var err error
	var middleware Middleware

	if middleware, err = middlewareFactory(f.ctx, f.config, f.logger); err != nil {
		return fmt.Errorf("can not create middleware: %w", err)
	}

	if position == PositionBeginning {
		f.middlewares = append([]Middleware{middleware}, f.middlewares...)
	} else {
		f.middlewares = append(f.middlewares, middleware)
	}

	return nil
}

func (f *factory) addModuleToStage(name string, module Module, opts []ModuleOption) error {
	ms := &moduleState{
		module:    module,
		config:    getModuleConfig(module),
		isRunning: 0,
		err:       nil,
	}

	MergeOptions(opts)(&ms.config)

	var ok bool
	var stage *stage

	// if the module specified a stage we do not yet have we have to add a new stage.
	if stage, ok = f.stages[ms.config.stage]; !ok {
		stage = f.newStage(ms.config.stage)
	}

	if _, didExist := stage.modules.modules[name]; didExist {
		// if we overwrite an existing module, the module count will be off and the application will hang while waiting
		// until stage.moduleCount modules have booted.
		return fmt.Errorf("failed to add new module %s: module exists", name)
	}

	stage.modules.modules[name] = ms

	return nil
}

func (f *factory) newStage(index int) *stage {
	s := newStage(f.ctx, f.config, f.logger, index)
	f.stages[index] = s

	return s
}
