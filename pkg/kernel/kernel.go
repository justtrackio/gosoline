package kernel

import (
	"context"
	"errors"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/coffin"
	"github.com/applike/gosoline/pkg/mon"
	"os"
	"os/signal"
	"sort"
	"sync"
	"syscall"
	"time"
)

type kernel struct {
	cfn coffin.Coffin
	sig chan os.Signal
	lck sync.Mutex
	wg  sync.WaitGroup

	config cfg.Config
	logger mon.Logger

	factories   []ModuleFactory
	modules     sync.Map
	moduleCount int
	isRunning   bool
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
	k.debugConfig()

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

	if bootErr != nil {
		k.logger.Error(bootErr, "error during the boot process of the kernel")
		return
	}

	var ctx context.Context
	k.cfn, ctx = coffin.WithContext(context.Background())
	k.wg.Add(k.moduleCount)

	k.modules.Range(func(name interface{}, moduleState interface{}) bool {
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

	select {
	case <-ctx.Done():
		k.Stop("context done")
	case sig := <-k.sig:
		reason := fmt.Sprintf("signal %s", sig.String())
		k.Stop(reason)
	}

	go func() {
		timer := time.NewTimer(10 * time.Second)
		<-timer.C

		err := errors.New("kernel was not able to shutdown in 10 seconds")
		k.logger.Error(err, "kernel shutdown seems to be blocking.. exiting...")

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

func (k *kernel) runFactories() error {
	defer k.recover("error running module factories")

	for _, f := range k.factories {
		modules, err := f(k.config, k.logger)

		if err != nil {
			return err
		}

		for name, m := range modules {
			k.Add(name, m)
		}
	}

	return nil
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

func (k *kernel) recover(msg string) {
	err := coffin.ResolveRecovery(recover())

	if err != nil {
		k.logger.Error(err.(error), msg)
	}
}

func (k *kernel) getModuleState(name string) *ModuleState {
	ms, ok := k.modules.Load(name)

	if !ok {
		panic(errors.New(fmt.Sprintf("module %v not found", name)))
	}

	return ms.(*ModuleState)
}

func (k *kernel) debugConfig() {
	keys := k.config.AllKeys()
	sort.Strings(keys)

	for _, key := range keys {
		k.logger.Infof("cfg %v=%v", key, k.config.Get(key))
	}
}
