package kernel

import (
	"context"
	"fmt"
	"sync"

	"github.com/justtrackio/gosoline/pkg/coffin"
	"github.com/justtrackio/gosoline/pkg/conc"
	"github.com/justtrackio/gosoline/pkg/log"
)

var ErrKernelStopping = fmt.Errorf("stopping kernel")

type stage struct {
	lck sync.Mutex
	cfn coffin.Coffin
	err error

	terminated conc.SignalOnce

	modules modules
}

type modules struct {
	lck     conc.PoisonedLock
	modules map[string]*ModuleState
}

func newStage() *stage {
	return &stage{
		terminated: conc.NewSignalOnce(),

		modules: modules{
			lck:     conc.NewPoisonedLock(),
			modules: make(map[string]*ModuleState),
		},
	}
}

func (s *stage) run(k *kernel) {
	if err := s.modules.lck.Poison(); err != nil {
		k.logger.Error("stage was already run: %w", err)
		return
	}

	s.lck.Lock()
	defer s.lck.Unlock()

	s.cfn = coffin.WithContext(k.ctx, func(cfn coffin.StartingCoffin, ctx context.Context) {
		for name, ms := range s.modules.modules {
			cfn.Gof(func(name string, ms *ModuleState) func() error {
				return func() error {
					return k.runModule(ctx, name, ms)
				}
			}(name, ms), "panic during running of module %s", name)
		}
	})
}

func (s *stage) stopWait(stageIndex int, logger log.Logger) {
	s.lck.Lock()
	cfn := s.cfn
	s.lck.Unlock()

	if cfn != nil {
		cfn.Kill(ErrKernelStopping)
		s.err = cfn.Wait()
	} else {
		s.err = fmt.Errorf("can not stop stage which is not yet running")
	}

	if s.err != nil && s.err != ErrKernelStopping {
		logger.Error("error during the execution of stage %d: %w", stageIndex, s.err)
	}

	s.terminated.Signal()
}

func (s *stage) waitStopping() {
	s.lck.Lock()
	cfn := s.cfn
	s.lck.Unlock()

	<-cfn.Dying()
}
