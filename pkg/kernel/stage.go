package kernel

import (
	"context"
	"fmt"
	"sort"

	"github.com/justtrackio/gosoline/pkg/coffin"
	"github.com/justtrackio/gosoline/pkg/conc"
	"github.com/justtrackio/gosoline/pkg/log"
)

var ErrKernelStopping = fmt.Errorf("stopping kernel")

type modules struct {
	lck     conc.PoisonedLock
	modules map[string]*ModuleState
}

func (m modules) len() int {
	return len(m.modules)
}

type stage struct {
	cfn coffin.Coffin
	ctx context.Context
	err error

	running    conc.SignalOnce
	terminated conc.SignalOnce

	modules modules
}

func newStage(ctx context.Context) *stage {
	cfn, ctx := coffin.WithContext(ctx)

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
	if err := s.modules.lck.Poison(); err != nil {
		k.logger.Error("stage was already run: %w", err)
		return
	}

	for name, ms := range s.modules.modules {
		s.cfn.Gof(func(name string, ms *ModuleState) func() error {
			return func() error {
				// wait until every routine of the stage was spawned
				// if a module exists too fast, we have a race condition
				// regarding the precondition of tomb.Go (namely that no
				// new routine may be added after the last one exited)
				<-s.running.Channel()

				resultErr := k.runModule(s.ctx, name, ms)

				if resultErr != nil {
					k.Stop(fmt.Sprintf("module %s returned with an error", name))
				}

				return resultErr
			}
		}(name, ms), "panic during running of module %s", name)
	}

	s.running.Signal()
}

func (s *stage) stopWait(stageIndex int, logger log.Logger) {
	s.cfn.Kill(ErrKernelStopping)
	s.err = s.cfn.Wait()

	if s.err != nil && s.err != ErrKernelStopping {
		logger.Error("error during the execution of stage %d: %w", stageIndex, s.err)
	}

	s.terminated.Signal()
}

func (s *stage) len() int {
	return s.modules.len()
}

type stages map[int]*stage

func (s stages) hasModules() bool {
	// no need to iterate in order as we are only checking
	for _, stage := range s {
		if len(stage.modules.modules) > 0 {
			return true
		}
	}

	return false
}

func (s stages) countForegroundModules() int32 {
	count := int32(0)

	// no need to iterate in order as we are only counting
	for _, stage := range s {
		for _, m := range stage.modules.modules {
			if !m.Config.Background {
				count++
			}
		}
	}

	return count
}

func (s stages) getIndices() []int {
	keys := make([]int, len(s))
	i := 0

	for k := range s {
		keys[i] = k
		i++
	}

	sort.Ints(keys)

	return keys
}
