package kernel

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/coffin"
	"github.com/justtrackio/gosoline/pkg/conc"
	"github.com/justtrackio/gosoline/pkg/log"
	"golang.org/x/sys/unix"
)

const (
	ExitCodeOk           = 0
	ExitCodeErr          = 1
	ExitCodeNothingToRun = 10
	ExitCodeNoForeground = 11
	ExitCodeForced       = 12
)

type ExitHandler func(code int)

//go:generate mockery --name Kernel
type Kernel interface {
	Add(name string, moduleFactory ModuleFactory, opts ...ModuleOption)
	AddFactory(factory MultiModuleFactory)
	AddMiddleware(middleware Middleware, position Position)
	Running() <-chan struct{}
	Run()
	Stop(reason string)
}

type Option func(k *kernel) error

type GosoKernel interface {
	Kernel
	Option(options ...Option) error
}

type kernel struct {
	config cfg.Config
	logger log.Logger

	moduleSetupContainers []moduleSetupContainer
	multiFactories        []MultiModuleFactory
	middlewares           []Middleware

	ctx               context.Context
	stages            map[int]*stage
	stagesLck         conc.PoisonedLock
	started           conc.PoisonedLock
	running           chan struct{}
	stopped           sync.Once
	foregroundModules int32

	killTimeout time.Duration
	exitCode    int
	exitOnce    sync.Once
	exitHandler ExitHandler
}

func New(ctx context.Context, config cfg.Config, logger log.Logger, options ...Option) (*kernel, error) {
	k := &kernel{
		config: config,
		logger: logger.WithChannel("kernel"),

		moduleSetupContainers: make([]moduleSetupContainer, 0),
		multiFactories:        make([]MultiModuleFactory, 0),

		ctx:       ctx,
		stages:    make(map[int]*stage),
		stagesLck: conc.NewPoisonedLock(),
		running:   make(chan struct{}),
		started:   conc.NewPoisonedLock(),

		killTimeout: time.Second * 10,
		exitCode:    ExitCodeErr,
		exitHandler: os.Exit,
	}

	if err := k.Option(options...); err != nil {
		return nil, fmt.Errorf("failed to configure kernel: %w", err)
	}

	return k, nil
}

func (k *kernel) Option(options ...Option) error {
	if err := k.started.TryLock(); err != nil {
		return fmt.Errorf("kernel already running: %w", err)
	}
	defer k.started.Unlock()

	for _, opt := range options {
		if err := opt(k); err != nil {
			return err
		}
	}

	return nil
}

func (k *kernel) Add(name string, moduleFactory ModuleFactory, opts ...ModuleOption) {
	container := moduleSetupContainer{
		name:    name,
		factory: moduleFactory,
		opts:    opts,
	}

	k.moduleSetupContainers = append(k.moduleSetupContainers, container)
}

func (k *kernel) AddFactory(factory MultiModuleFactory) {
	k.multiFactories = append(k.multiFactories, factory)
}

func (k *kernel) AddMiddleware(middleware Middleware, position Position) {
	if position == PositionBeginning {
		k.middlewares = append([]Middleware{middleware}, k.middlewares...)
	} else {
		k.middlewares = append(k.middlewares, middleware)
	}
}

// Run will boot and run the modules added to the kernel.
// By default, os.Exit will get called if an error occurs or after the modules have stopped running,
// which means that there will be no return out of this call.
func (k *kernel) Run() {
	// do not allow config changes anymore
	if err := k.started.Poison(); err != nil {
		k.logger.Error("kernel was already run: %w", err)
		// don't close running here, it should've been already closed when the lock was already poisoned
		return
	}

	defer k.exit()
	k.logger.Info("starting kernel")

	if err := k.runMultiFactories(k.ctx); err != nil {
		k.logger.Error("error building additional modules by multiFactories: %w", err)
		close(k.running)
		k.exitCode = ExitCodeErr
		return
	}

	if len(k.moduleSetupContainers) == 0 {
		k.logger.Warn("nothing to run")
		close(k.running)
		k.exitCode = ExitCodeNothingToRun
		return
	}

	if err := k.runFactories(k.ctx); err != nil {
		k.logger.Error("error building modules: %w", err)
		close(k.running)
		k.exitCode = ExitCodeErr
		return
	}

	k.logger.Info("all modules created")

	// poison our stages so any other thread trying to add a new stage will
	// panic instead of hanging
	if err := k.stagesLck.Poison(); err != nil {
		k.logger.Error("stages have already been run: %w", err)
		close(k.running)
		return
	}

	if !k.hasModules() {
		close(k.running)
		k.logger.Warn("nothing to run")
		k.exitCode = ExitCodeNothingToRun
		return
	}

	k.foregroundModules = int32(k.countForegroundModules())
	if k.foregroundModules == 0 {
		k.logger.Warn("no foreground modules")
		k.exitCode = ExitCodeNoForeground
		return
	}

	k.debugConfig()

	runHandler := func() {
		sig := make(chan os.Signal, 2)
		signal.Notify(sig, unix.SIGTERM, unix.SIGINT)

		for _, stageIndex := range k.getStageIndices() {
			k.stages[stageIndex].run(k)
			k.logger.Info("stage %d up and running", stageIndex)
		}

		k.logger.Info("kernel up and running")
		close(k.running)

		select {
		case <-k.waitAllStagesStopping().Channel():
			k.Stop("context done")
		case sig := <-sig:
			reason := fmt.Sprintf("signal %s", sig.String())
			k.Stop(reason)
		}

		k.waitStopped()

		hasErr := false
		for _, stage := range k.stages {
			if stage.err != nil && stage.err != ErrKernelStopping {
				hasErr = true
			}
		}

		if !hasErr {
			k.exitCode = ExitCodeOk
		}
	}

	for i := len(k.middlewares) - 1; i >= 0; i-- {
		runHandler = k.middlewares[i](k.ctx, k.config, k.logger, runHandler)
	}

	runHandler()
}

func (k *kernel) Stop(reason string) {
	k.stopped.Do(func() {
		go func() {
			k.logger.Info("stopping kernel due to: %s", reason)
			indices := k.getStageIndices()

			for i := len(indices) - 1; i >= 0; i-- {
				stageIndex := indices[i]
				k.logger.Info("stopping stage %d", stageIndex)
				k.stages[stageIndex].stopWait(stageIndex, k.logger)
				k.logger.Info("stopped stage %d", stageIndex)
			}
		}()
	})
}

func (k *kernel) Running() <-chan struct{} {
	return k.running
}

func (k *kernel) exit() {
	k.exitOnce.Do(func() {
		k.logger.Info("leaving kernel with exit code %d", k.exitCode)
		k.exitHandler(k.exitCode)
	})
}

func (k *kernel) runMultiFactories(ctx context.Context) (err error) {
	defer func() {
		if err != nil {
			return
		}

		err = coffin.ResolveRecovery(recover())
	}()

	var moduleFactories map[string]ModuleFactory

	for _, factory := range k.multiFactories {
		if moduleFactories, err = factory(ctx, k.config, k.logger); err != nil {
			return err
		}

		for name, m := range moduleFactories {
			k.Add(name, m)
		}
	}

	return
}

func (k *kernel) runFactories(ctx context.Context) error {
	bootCoffin := coffin.New(func(bootCoffin coffin.StartingCoffin) {
		bookLck := sync.Mutex{}

		for _, container := range k.moduleSetupContainers {
			bootCoffin.GoWithContextf(ctx, func(container moduleSetupContainer) func(ctx context.Context) error {
				return func(ctx context.Context) error {
					module, err := container.factory(ctx, k.config, k.logger)
					if err != nil {
						return fmt.Errorf("can not build module %s: %w", container.name, err)
					}

					bookLck.Lock()
					defer bookLck.Unlock()

					if err = k.addModuleToStage(container.name, module, container.opts); err != nil {
						return fmt.Errorf("can not add module to stage: %w", err)
					}

					return nil
				}
			}(container), "panic during boot of module %s", container.name)
		}
	})

	return bootCoffin.Wait()
}

func (k *kernel) addModuleToStage(name string, module Module, opts []ModuleOption) error {
	ms := &ModuleState{
		Module: module,
		Config: getModuleConfig(module),
	}

	MergeOptions(opts)(&ms.Config)

	// lock the stagesLck even if we are just reading from the map
	// we are not allowed to read and write a map concurrently
	k.stagesLck.MustLock()

	stage, ok := k.stages[ms.Config.Stage]

	// if the module specified a stage we do not yet have we have to add a new stage.
	if !ok {
		stage = k.newStage(ms.Config.Stage)
	}

	k.stagesLck.Unlock()

	if err := stage.modules.lck.TryLock(); err != nil {
		return fmt.Errorf("failed to add new module %s: kernel is already running. You have to add your modules before running the kernel: %w", name, err)
	}
	defer stage.modules.lck.Unlock()

	if _, didExist := stage.modules.modules[name]; didExist {
		// if we overwrite an existing module, the module count will be off and the application will hang while waiting
		// until stage.moduleCount modules have booted.
		return fmt.Errorf("failed to add new module %s: module exists", name)
	}

	stage.modules.modules[name] = ms

	return nil
}

func (k *kernel) newStage(index int) *stage {
	s := newStage()
	k.stages[index] = s

	return s
}

func (k *kernel) hasModules() bool {
	// no need to iterate in order as we are only checking
	for _, stage := range k.stages {
		if len(stage.modules.modules) > 0 {
			return true
		}
	}

	return false
}

func (k *kernel) countForegroundModules() int {
	count := 0

	// no need to iterate in order as we are only counting
	for _, stage := range k.stages {
		for _, m := range stage.modules.modules {
			if !m.Config.Background {
				count++
			}
		}
	}

	return count
}

func (k *kernel) debugConfig() {
	debugErr := cfg.DebugConfig(k.config, k.logger)

	if debugErr != nil {
		k.logger.Error("can not debug config: %w", debugErr)
	}
}

func (k *kernel) runModule(ctx context.Context, name string, ms *ModuleState) (moduleErr error) {
	defer k.logger.Info("stopped %s module %s", ms.Config.GetType(), name)

	k.logger.Info("running %s module %s in stage %d", ms.Config.GetType(), name, ms.Config.Stage)

	ms.isRunning.Set(true)

	defer func(ms *ModuleState) {
		// recover any crash from the module - if we let the coffin handle this,
		// this is already too late because we might have killed the kernel and
		// swallowed the error
		panicErr := coffin.ResolveRecovery(recover())

		if panicErr != nil {
			moduleErr = panicErr
		}

		if moduleErr != nil {
			k.logger.Error("error running %s module %s: %w", ms.Config.GetType(), name, moduleErr)
		}

		ms.isRunning.Set(false)
		if ms.Config.Essential {
			k.essentialModuleExited(name)
		} else if !ms.Config.Background {
			k.foregroundModuleExited()
		}
	}(ms)

	return ms.Module.Run(ctx)
}

func (k *kernel) essentialModuleExited(name string) {
	// actually we would need to decrement k.foregroundModules here, too
	// however, as we are stopping in any case, we don't have to
	reason := fmt.Sprintf("the essential module [%s] has stopped running", name)
	k.Stop(reason)
}

func (k *kernel) foregroundModuleExited() {
	remaining := atomic.AddInt32(&k.foregroundModules, -1)

	if remaining == 0 {
		k.Stop("no more foreground modules in running state")
	}
}

func (k *kernel) waitStopped() {
	done := conc.NewSignalOnce()
	defer done.Signal()

	go func() {
		timer := time.NewTimer(k.killTimeout)
		defer timer.Stop()

		select {
		case <-timer.C:
			err := fmt.Errorf("kernel was not able to shutdown in %v", k.killTimeout)
			k.logger.Error("kernel shutdown seems to be blocking.. exiting...: %w", err)

			// we don't need to iterate in order, but doing so is much nicer, so lets do it
			for _, stageIndex := range k.getStageIndices() {
				s := k.stages[stageIndex]
				for name, ms := range s.modules.modules {
					if ms.isRunning.Get() {
						k.logger.Info("module in stage %d blocking the shutdown: %s", stageIndex, name)
					}
				}
			}

			k.exitCode = ExitCodeForced
			k.exit()
		case <-done.Channel():
			return
		}
	}()

	// we don't need to iterate in order, we just need to block until everything is done
	for _, stage := range k.stages {
		<-stage.terminated.Channel()
	}
}

func (k *kernel) getStageIndices() []int {
	keys := make([]int, len(k.stages))
	i := 0

	for k := range k.stages {
		keys[i] = k
		i++
	}

	sort.Ints(keys)

	return keys
}

func (k *kernel) waitAllStagesStopping() conc.SignalOnce {
	done := conc.NewSignalOnce()

	go func() {
		for _, s := range k.stages {
			s.waitStopping()
		}

		done.Signal()
	}()

	return done
}
