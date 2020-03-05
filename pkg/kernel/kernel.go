package kernel

import (
	"context"
	"errors"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/coffin"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/jeremywohl/flatten"
	"github.com/thoas/go-funk"
	"os"
	"os/signal"
	"sort"
	"sync"
	"syscall"
	"time"
)

type Settings struct {
	KillTimeout time.Duration `cfg:"killTimeout" default:"10s"`
}

//go:generate mockery -name=Kernel
type Kernel interface {
	Add(name string, module Module)
	AddFactory(factory ModuleFactory)
	Booted() <-chan struct{}
	Running() <-chan struct{}
	Run()
	Stop(reason string)
}

type kernel struct {
	config cfg.Config
	logger mon.Logger

	cfn     coffin.Coffin
	sig     chan os.Signal
	booted  chan struct{}
	running chan struct{}
	lck     sync.Mutex
	wg      sync.WaitGroup

	settings    *Settings
	factories   []ModuleFactory
	modules     sync.Map
	moduleCount int
	isRunning   bool
}

func New(config cfg.Config, logger mon.Logger, settings *Settings) Kernel {
	return &kernel{
		sig: make(chan os.Signal, 1),

		booted:  make(chan struct{}),
		running: make(chan struct{}),

		config: config,
		logger: logger.WithChannel("kernel"),

		settings:  settings,
		factories: make([]ModuleFactory, 0),
	}
}

// Booted channel will be closed as soon as all modules Boot functions were executed
func (k *kernel) Booted() <-chan struct{} {
	return k.booted
}

// Running channel will be closed as soon as all modules Run functions were invoked
func (k *kernel) Running() <-chan struct{} {
	return k.running
}

func (k *kernel) Add(name string, module Module) {
	state := &ModuleState{
		Module:    module,
		IsRunning: false,
	}

	k.modules.Store(name, state)
	k.moduleCount++
}

func (k *kernel) AddFactory(factory ModuleFactory) {
	k.factories = append(k.factories, factory)
}

func (k *kernel) Run() {
	defer k.logger.Info("leaving kernel")
	k.logger.Info("starting kernel")

	err := k.runFactories()
	if err != nil {
		k.logger.Error(err, "error building additional modules by factories")
		return
	}

	if k.moduleCount == 0 {
		k.logger.Info("nothing to run")
		return
	}

	if !k.hasForegroundModules() {
		k.logger.Info("no foreground modules")
		return
	}

	signal.Notify(k.sig, syscall.SIGTERM)
	signal.Notify(k.sig, syscall.SIGINT)

	bootCoffin := coffin.New()

	k.modules.Range(func(name interface{}, moduleState interface{}) bool {
		bootCoffin.Gof(func() error {
			return k.bootModule(name.(string))
		}, "panic during boot of module %s", name)

		return true
	})

	bootErr := bootCoffin.Wait()
	debugErr := k.debugConfig()

	if debugErr != nil {
		k.logger.Error(bootErr, "can not debug config")
		return
	}

	if bootErr != nil {
		k.logger.Error(bootErr, "error during the boot process of the kernel")
		return
	}

	k.logger.Info("all modules booted")
	close(k.booted)

	var ctx context.Context
	k.cfn, ctx = coffin.WithContext(context.Background())
	k.wg.Add(k.moduleCount)

	k.modules.Range(func(name interface{}, moduleState interface{}) bool {
		// TODO: gosoline#201 THIS IS EXECUTED ASYNCHRONOUSLY! MODULES ARE NOT YET RUNNING AFTER Range HAS EXECUTED!
		k.cfn.Gof(func() error {
			err := k.runModule(name.(string), ctx)
			k.checkRunningForegroundModules()

			return err
		}, "panic during running of module %s", name)

		return true
	})

	k.checkRunningEssentialModules()
	k.isRunning = true
	k.logger.Info("kernel up and running")
	close(k.running)

	select {
	case <-ctx.Done():
		k.Stop("context done")
	case sig := <-k.sig:
		reason := fmt.Sprintf("signal %s", sig.String())
		k.Stop(reason)
	}

	go func() {
		timer := time.NewTimer(k.settings.KillTimeout)
		<-timer.C

		err := fmt.Errorf("kernel was not able to shutdown in %v", k.settings.KillTimeout)
		k.logger.Error(err, "kernel shutdown seems to be blocking.. exiting...")

		k.modules.Range(func(name interface{}, moduleState interface{}) bool {
			ms := moduleState.(*ModuleState)
			if ms.IsRunning {
				k.logger.Infof("module blocking the shutdown: %s", name)
			}

			return true
		})

		os.Exit(1)
	}()

	err = k.cfn.Wait()

	if err != nil {
		k.logger.Error(err, "error during the execution of the kernel")
	}
}

func (k *kernel) Stop(reason string) {
	k.isRunning = false
	k.logger.Infof("stopping kernel due to: %s", reason)
	k.cfn.Kill(nil)
}

func (k *kernel) runFactories() (err error) {
	defer func() {
		err = coffin.ResolveRecovery(recover())

		if err != nil {
			k.logger.Error(err, "error running module factories")
		}
	}()

	for _, f := range k.factories {
		modules, err := f(k.config, k.logger)

		if err != nil {
			return err
		}

		for name, m := range modules {
			k.Add(name, m)
		}
	}

	return
}

func (k *kernel) bootModule(name string) error {
	k.logger.Infof("booting module %s", name)
	ms := k.getModuleState(name)

	logger := k.logger.WithChannel("default")
	err := ms.Module.Boot(k.config, logger)
	ms.Err = err

	k.logger.Infof("booted module %s", name)

	return err
}

func (k *kernel) runModule(name string, ctx context.Context) error {
	defer k.logger.Info("stopped module " + name)

	k.logger.Info("running module " + name)

	ms := k.getModuleState(name)
	ms.IsRunning = true
	k.wg.Done()

	defer func(ms *ModuleState) {
		ms.IsRunning = false
	}(ms)
	err := ms.Module.Run(ctx)

	if err != nil {
		k.logger.Error(err, "error running module "+name)
	}
	ms.Err = err

	return err
}

func (k *kernel) hasForegroundModules() bool {
	hasForegroundModule := false

	k.modules.Range(func(name interface{}, moduleState interface{}) bool {
		ms := moduleState.(*ModuleState)

		if isForegroundModule(ms.Module) {
			hasForegroundModule = true
			return false
		}

		return true
	})

	return hasForegroundModule
}

func (k *kernel) checkRunningEssentialModules() {
	go func() {
		for {
			time.Sleep(time.Second)
			k.doCheckRunningEssentialModules()
		}
	}()
}

func (k *kernel) doCheckRunningEssentialModules() {
	if !k.isRunning {
		return
	}

	k.wg.Wait()

	var reason string
	hasEssentialModuleStopped := false

	k.modules.Range(func(name interface{}, moduleState interface{}) bool {
		ms := moduleState.(*ModuleState)
		rt := ms.Module.GetType()

		if !ms.IsRunning && rt == TypeEssential {
			reason = fmt.Sprintf("the essential module [%s] has stopped running", name)
			hasEssentialModuleStopped = true
			return false
		}

		return true
	})

	if !hasEssentialModuleStopped {
		return
	}

	k.Stop(reason)
}

func (k *kernel) checkRunningForegroundModules() {
	if !k.isRunning {
		return
	}

	k.wg.Wait()

	k.lck.Lock()
	defer k.lck.Unlock()

	hasForegroundModuleRunning := false

	k.modules.Range(func(name interface{}, moduleState interface{}) bool {
		ms := moduleState.(*ModuleState)

		if ms.IsRunning && isForegroundModule(ms.Module) {
			hasForegroundModuleRunning = true
			return false
		}

		return true
	})

	if hasForegroundModuleRunning {
		return
	}

	k.Stop("no more foreground modules in running state")
}

func (k *kernel) getModuleState(name string) *ModuleState {
	ms, ok := k.modules.Load(name)

	if !ok {
		panic(errors.New(fmt.Sprintf("module %v not found", name)))
	}

	return ms.(*ModuleState)
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
