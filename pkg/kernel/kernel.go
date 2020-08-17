package kernel

import (
	"context"
	"errors"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/coffin"
	"github.com/applike/gosoline/pkg/conc"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/jeremywohl/flatten"
	"github.com/thoas/go-funk"
	"golang.org/x/sys/unix"
	"os"
	"os/signal"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

//go:generate mockery -name=Kernel
type Kernel interface {
	Add(name string, module Module, opts ...ModuleOption)
	AddFactory(factory ModuleFactory)
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
	logger mon.Logger

	stages            map[int]*stage
	stagesLck         conc.PoisonedLock
	factories         []ModuleFactory
	started           conc.PoisonedLock
	running           chan struct{}
	stopped           sync.Once
	foregroundModules int32

	killTimeout time.Duration
	forceExit   func(code int)
}

func New(config cfg.Config, logger mon.Logger, options ...Option) *kernel {
	k := &kernel{
		stages:    make(map[int]*stage),
		stagesLck: conc.NewPoisonedLock(),
		factories: make([]ModuleFactory, 0),
		running:   make(chan struct{}),
		started:   conc.NewPoisonedLock(),

		config: config,
		logger: logger.WithChannel("kernel"),

		killTimeout: time.Second * 10,
		forceExit:   os.Exit,
	}

	if err := k.Option(options...); err != nil {
		logger.Panic(err, "failed to configure kernel")
	}

	return k
}

func KillTimeout(killTimeout time.Duration) Option {
	return func(k *kernel) error {
		k.killTimeout = killTimeout

		return nil
	}
}

func ForceExit(forceExit func(code int)) Option {
	return func(k *kernel) error {
		k.forceExit = forceExit

		return nil
	}
}

func (k *kernel) newStage(index int) *stage {
	s := newStage()
	k.stages[index] = s

	return s
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

func (k *kernel) Add(name string, module Module, opts ...ModuleOption) {
	ms := &ModuleState{
		Module:    module,
		Config:    getModuleConfig(module),
		IsRunning: false,
		Err:       nil,
	}

	MergeOptions(opts)(&ms.Config)

	// lock the stagesLck even if we are just reading from the map
	// we are not allowed to read and write a map concurrently
	k.stagesLck.Lock()

	stage, ok := k.stages[ms.Config.Stage]

	// if the module specified a stage we do not yet have we have to add a new stage.
	if !ok {
		stage = k.newStage(ms.Config.Stage)
	}

	k.stagesLck.Unlock()

	if err := stage.modules.lck.TryLock(); err != nil {
		k.logger.Panicf(
			err,
			"Failed to add new module %s: kernel is already running. You have to add your modules before running the kernel",
			name,
		)
	}
	defer stage.modules.lck.Unlock()

	if _, didExist := stage.modules.modules[name]; didExist {
		// if we overwrite an existing module, the module count will be off and the application will hang while waiting
		// until stage.moduleCount modules have booted.
		k.logger.Panicf(
			errors.New("module must not be redeclared"),
			"failed to add new module %s: module exists",
			name,
		)
	}

	stage.modules.modules[name] = ms
}

func (k *kernel) AddFactory(factory ModuleFactory) {
	k.factories = append(k.factories, factory)
}

func (k *kernel) Running() <-chan struct{} {
	return k.running
}

func (k *kernel) Run() {
	// do not allow config changes anymore
	k.started.Poison()

	defer k.logger.Info("leaving kernel")
	k.logger.Info("starting kernel")

	if err := k.runFactories(); err != nil {
		k.logger.Error(err, "error building additional modules by factories")
		close(k.running)
		return
	}

	// poison our stages so any other thread trying to add a new stage will
	// panic instead of hanging
	k.stagesLck.Poison()

	if !k.hasModules() {
		close(k.running)
		k.logger.Info("nothing to run")
		return
	}

	k.foregroundModules = int32(k.countForegroundModules())
	if k.foregroundModules == 0 {
		k.logger.Info("no foreground modules")
		return
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, unix.SIGTERM)
	signal.Notify(sig, unix.SIGINT)

	if !k.boot() {
		return
	}

	for _, stageIndex := range k.getStageIndices() {
		k.stages[stageIndex].run(k)
		k.logger.Infof("stage %d up and running", stageIndex)
	}

	k.logger.Info("kernel up and running")
	close(k.running)

	select {
	case <-k.waitAllStagesDone().Channel():
		k.Stop("context done")
	case sig := <-sig:
		reason := fmt.Sprintf("signal %s", sig.String())
		k.Stop(reason)
	}

	k.waitStopped()
}

func (k *kernel) Stop(reason string) {
	k.stopped.Do(func() {
		go func() {
			k.logger.Infof("stopping kernel due to: %s", reason)
			indices := k.getStageIndices()
			for i := len(indices) - 1; i >= 0; i-- {
				stageIndex := indices[i]
				k.logger.Infof("stopping stage %d", stageIndex)
				// wait until the stage was at least booted. Otherwise we might kill a stage before
				// it is fully initialized
				<-k.stages[stageIndex].booted.Channel()
				k.stages[stageIndex].stopWait(stageIndex, k.logger)
				k.logger.Infof("stopped stage %d", stageIndex)
			}
		}()
	})
}

func (k *kernel) runFactories() (err error) {
	defer func() {
		if err != nil {
			return
		}

		if err = coffin.ResolveRecovery(recover()); err != nil {
			k.logger.Error(err, "error running module factories")
		}
	}()

	var modules map[string]Module

	for _, factory := range k.factories {
		if modules, err = factory(k.config, k.logger); err != nil {
			return
		}

		for name, m := range modules {
			k.Add(name, m)
		}
	}

	return
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
			if m.Config.Type != TypeBackground {
				count++
			}
		}
	}

	return count
}

func (k *kernel) boot() bool {
	// boot all stages in ascending order, starting with the essential stage
	var bootErr error
	for _, stageIndex := range k.getStageIndices() {
		stage := k.stages[stageIndex]
		stage.prepare()
		k.logger.Infof("booting stage %d", stageIndex)

		if bootErr == nil {
			bootErr = stage.boot(k)
		}

		stage.booted.Signal()
	}

	debugErr := k.debugConfig()

	if debugErr != nil {
		k.logger.Error(bootErr, "can not debug config")
		return false
	}

	if bootErr != nil {
		k.logger.Error(bootErr, "error during the boot process of the kernel")
		return false
	}

	k.logger.Info("all modules booted")

	return true
}

func (k *kernel) debugConfig() error {
	settings := k.config.AllSettings()
	flattened, err := flatten.Flatten(settings, "", flatten.DotStyle)

	if err != nil {
		return fmt.Errorf("can not flatten config settings")
	}

	keys := funk.Keys(flattened).([]string)
	sort.Strings(keys)

	for _, key := range keys {
		k.logger.Infof("cfg %v=%v", key, flattened[key])
	}

	return nil
}

func (k *kernel) runModule(name string, ms *ModuleState, ctx context.Context) error {
	defer k.logger.Infof("stopped %s module %s", ms.Config.Type, name)

	k.logger.Infof("running %s module %s", ms.Config.Type, name)

	ms.IsRunning = true

	defer func(ms *ModuleState) {
		ms.IsRunning = false
		switch ms.Config.Type {
		case TypeEssential:
			k.essentialModuleExited(name)
		case TypeForeground:
			k.foregroundModuleExited()
		}
	}(ms)
	ms.Err = ms.Module.Run(ctx)

	if ms.Err != nil {
		k.logger.Errorf(ms.Err, "error running %s module %s", ms.Config.Type, name)
	}

	return ms.Err
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
			k.logger.Errorf(err, "kernel shutdown seems to be blocking.. exiting...")

			// we don't need to iterate in order, but doing so is much nicer, so lets do it
			for _, stageIndex := range k.getStageIndices() {
				s := k.stages[stageIndex]
				for name, ms := range s.modules.modules {
					if ms.IsRunning {
						k.logger.Infof("module in stage %d blocking the shutdown: %s", stageIndex, name)
					}
				}
			}

			k.forceExit(1)
		case <-done.Channel():
			return
		}
	}()

	// we don't need to iterate in order, we just need to block until everything is done
	for _, stage := range k.stages {
		_ = <-stage.terminated.Channel()
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

func (k *kernel) waitAllStagesDone() conc.SignalOnce {
	done := conc.NewSignalOnce()
	wg := &sync.WaitGroup{}
	wg.Add(len(k.stages))

	for _, s := range k.stages {
		go func(ctx context.Context) {
			_ = <-ctx.Done()
			wg.Done()
			wg.Wait()
			done.Signal()
		}(s.ctx)
	}

	return done
}
