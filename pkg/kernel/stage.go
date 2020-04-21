package kernel

import (
	"context"
	"errors"
	"fmt"
	"github.com/applike/gosoline/pkg/coffin"
	"github.com/applike/gosoline/pkg/mon"
)

var ErrKernelStopping = fmt.Errorf("stopping kernel")

type stage struct {
	cfn coffin.Coffin
	ctx context.Context

	booted     SignalOnce
	running    SignalOnce
	terminated SignalOnce

	modules modules
}

type modules struct {
	lck     PoisonedLock
	modules map[string]*ModuleState
}

func (s *stage) prepare() {
	s.modules.lck.Poison()
	s.cfn, s.ctx = coffin.WithContext(context.Background())
}

func (s *stage) boot(k *kernel) error {
	if len(s.modules.modules) == 0 {
		return errors.New("can not run empty stage")
	}

	bootCoffin := coffin.New()
	startBooting := NewSignalOnce()

	for name, m := range s.modules.modules {
		name := name
		m := m
		bootCoffin.Gof(func() error {
			// wait until we scheduled all boot routines
			// otherwise a fast booting module might violate the
			// condition of tomb.Go that no new routine must be
			// spawned after the last one exited
			<-startBooting.Channel()

			return bootModule(k, name, m)
		}, "panic during boot of module %s", name)
	}

	startBooting.Signal()

	return bootCoffin.Wait()
}

func (s *stage) run(k *kernel) {
	for name, m := range s.modules.modules {
		name := name
		m := m
		s.cfn.Gof(func() error {
			// wait until every routine of the stage was spawned
			// if a module exists too fast, we have a race condition
			// regarding the precondition of tomb.Go (namely that no
			// new routine may be added after the last one exited)
			<-s.running.Channel()

			return k.runModule(name, m, s.ctx)
		}, "panic during running of module %s", name)
	}

	s.running.Signal()
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
