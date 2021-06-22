package kernel

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/coffin"
	"github.com/applike/gosoline/pkg/conc"
	"github.com/applike/gosoline/pkg/log"
)

var ErrKernelStopping = fmt.Errorf("stopping kernel")

type stage struct {
	cfn coffin.Coffin
	ctx context.Context

	running    conc.SignalOnce
	terminated conc.SignalOnce

	modules modules
}

type modules struct {
	lck     conc.PoisonedLock
	modules map[string]*ModuleState
}

func newStage() *stage {
	cfn, ctx := coffin.WithContext(context.Background())

	return &stage{
		cfn: cfn,
		ctx: ctx,

		running:    conc.NewSignalOnce(),
		terminated: conc.NewSignalOnce(),

		modules: modules{
			lck:     conc.NewPoisonedLock(),
			modules: make(map[string]*ModuleState),
		},
	}
}

func (s *stage) run(k *kernel) {
	s.modules.lck.Poison()

	for name, ms := range s.modules.modules {
		s.cfn.Gof(func(name string, ms *ModuleState) func() error {
			return func() error {
				// wait until every routine of the stage was spawned
				// if a module exists too fast, we have a race condition
				// regarding the precondition of tomb.Go (namely that no
				// new routine may be added after the last one exited)
				<-s.running.Channel()

				return k.runModule(s.ctx, name, ms)
			}
		}(name, ms), "panic during running of module %s", name)
	}

	s.running.Signal()
}

func (s *stage) stopWait(stageIndex int, logger log.Logger) {
	s.cfn.Kill(ErrKernelStopping)
	err := s.cfn.Wait()

	if err != nil && err != ErrKernelStopping {
		logger.Error("error during the execution of stage %d: %w", stageIndex, err)
	}

	s.terminated.Signal()
}
