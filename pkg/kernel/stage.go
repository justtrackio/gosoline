package kernel

import (
	"errors"
	"fmt"
	"github.com/applike/gosoline/pkg/coffin"
	"github.com/applike/gosoline/pkg/mon"
)

var ErrKernelStopping = fmt.Errorf("stopping kernel")

type stage struct {
	cfn coffin.Coffin

	booted     SignalOnce
	running    SignalOnce
	terminated SignalOnce

	modules   modules
	isRunning bool
}

type modules struct {
	lck     PoisonedLock
	modules map[string]*ModuleState
}

func (s *stage) boot(k *kernel, bootCoffin coffin.Coffin) {
	s.modules.lck.Poison()

	if len(s.modules.modules) == 0 {
		// if we have no modules to boot, we need to run a dummy module - otherwise
		// our caller waits forever if the stage does not contain any module
		// so instead we run a single function which always fails and hopefully
		// alerts the caller that the stage is empty
		bootCoffin.Go(func() error {
			return errors.New("can not run empty stage")
		})
	}

	for name, m := range s.modules.modules {
		name := name
		m := m
		bootCoffin.Gof(func() error {
			return bootModule(k, name, m)
		}, "panic during boot of module %s", name)
	}
}

func (s *stage) stopWait(stageIndex int, logger mon.Logger) {
	s.cfn.Kill(ErrKernelStopping)
	err := s.cfn.Wait()

	if err != nil && err != ErrKernelStopping {
		logger.Errorf(err, "error during the execution of stage %d", stageIndex)
	}

	s.terminated.Signal()
}

func newStage() *stage {
	return &stage{
		booted:     NewSignalOnce(),
		running:    NewSignalOnce(),
		terminated: NewSignalOnce(),

		modules: modules{
			lck:     NewPoisonedLock(),
			modules: make(map[string]*ModuleState),
		},
	}
}

func bootModule(k *kernel, name string, ms *ModuleState) error {
	k.logger.Infof("booting module %s", name)

	logger := k.logger.WithChannel("default")
	ms.Err = ms.Module.Boot(k.config, logger)

	k.logger.Infof("booted module %s", name)

	return ms.Err
}
