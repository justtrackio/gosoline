package kernel

import (
	"context"
	"errors"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"gopkg.in/tomb.v2"
	"os"
	"os/signal"
	"reflect"
	"sort"
	"sync"
	"syscall"
	"time"
)

type kernel struct {
	t   *tomb.Tomb
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

	bootTomb := tomb.Tomb{}

	k.modules.Range(func(name interface{}, moduleState interface{}) bool {
		bootTomb.Go(func() error {
			booted, err := k.bootModule(name.(string))

			if err != nil {
				return err
			}

			if booted != true {
				return errors.New("could not boot module due to a panic")
			}

			return nil
		})

		return true
	})

	bootErr := bootTomb.Wait()

	if bootErr != nil {
		k.logger.Error(bootErr, "error during the boot process of the kernel")
		return
	}

	var ctx context.Context
	k.t, ctx = tomb.WithContext(context.Background())
	k.wg.Add(k.moduleCount)

	k.modules.Range(func(name interface{}, moduleState interface{}) bool {
		k.t.Go(func() error {
			err := k.runModule(name.(string), ctx)
			k.hasRunningForegroundModules()

			return err
		})

		return true
	})

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

	err = k.t.Wait()

	if err != nil {
		k.logger.Error(err, "error during the execution of the kernel")
	}
}

func (k *kernel) Stop(reason string) {
	k.isRunning = false
	k.logger.Infof("stopping kernel due to: %s", reason)
	k.t.Kill(nil)
}

func (k *kernel) runFactories() error {
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

func (k *kernel) bootModule(name string) (bool, error) {
	defer k.recover("error booting module " + name)

	k.logger.Infof("booting module %s", name)
	ms := k.getModuleState(name)

	logger := k.logger.WithChannel("default")
	err := ms.Module.Boot(k.config, logger)
	ms.Err = err

	k.logger.Infof("booted module %s", name)

	return true, err
}

func (k *kernel) runModule(name string, ctx context.Context) error {
	defer k.recover("error running module " + name)
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
		rt := ms.Module.GetType()

		if rt == TypeForeground {
			hasForegroundModule = true
			return false
		}

		return true
	})

	return hasForegroundModule
}

func (k *kernel) hasRunningForegroundModules() {
	if !k.isRunning {
		return
	}

	k.wg.Wait()

	k.lck.Lock()
	defer k.lck.Unlock()

	hasForegroundModuleRunning := false

	k.modules.Range(func(name interface{}, moduleState interface{}) bool {
		ms := moduleState.(*ModuleState)
		rt := ms.Module.GetType()

		if ms.IsRunning && rt == TypeForeground {
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
	err := recover()

	switch rval := err.(type) {
	case nil:
		return
	case error:
		k.logger.Error(rval, msg)
	case string:
		k.logger.Error(errors.New(err.(string)), msg)
	default:
		k.logger.Error(errors.New(fmt.Sprintf("unhandled error type %s", reflect.TypeOf(rval))), msg)
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
