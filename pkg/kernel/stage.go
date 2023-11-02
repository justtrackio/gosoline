package kernel

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/coffin"
	"github.com/justtrackio/gosoline/pkg/conc"
	"github.com/justtrackio/gosoline/pkg/log"
)

var ErrKernelStopping = fmt.Errorf("stopping kernel")

type modules struct {
	lck     conc.PoisonedLock
	modules map[string]*moduleState
}

func (m modules) len() int {
	return len(m.modules)
}

type stage struct {
	cfn                 coffin.Coffin
	ctx                 context.Context
	logger              log.Logger
	index               int
	healthCheckSettings HealthCheckSettings
	err                 error

	running    conc.SignalOnce
	terminated conc.SignalOnce

	modules modules
}

func newStage(ctx context.Context, config cfg.Config, logger log.Logger, index int) *stage {
	cfn, ctx := coffin.WithContext(ctx)

	settings := &Settings{}
	config.UnmarshalKey("kernel", settings)

	return &stage{
		cfn:                 cfn,
		ctx:                 ctx,
		logger:              logger,
		index:               index,
		healthCheckSettings: settings.HealthCheck,

		running:    conc.NewSignalOnce(),
		terminated: conc.NewSignalOnce(),

		modules: modules{
			lck:     conc.NewPoisonedLock(),
			modules: make(map[string]*moduleState),
		},
	}
}

func (s *stage) run(k *kernel) error {
	if err := s.modules.lck.Poison(); err != nil {
		return fmt.Errorf("stage was already run: %w", err)
	}

	for name, ms := range s.modules.modules {
		s.cfn.Gof(func(name string, ms *moduleState) func() error {
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

	return s.waitUntilHealthy()
}

func (s *stage) healthcheck() HealthCheckResult {
	var ok bool
	var err error
	var healthAware HealthCheckedModule
	var result HealthCheckResult

	for name, ms := range s.modules.modules {
		if healthAware, ok = ms.module.(HealthCheckedModule); !ok {
			continue
		}

		ok, err = healthAware.IsHealthy(s.ctx)

		result = append(result, ModuleHealthCheckResult{
			StageIndex: s.index,
			Name:       name,
			Healthy:    ok,
			Err:        err,
		})
	}

	return result
}

func (s *stage) waitUntilHealthy() error {
	var result HealthCheckResult

	timeoutTimer := clock.NewRealTimer(s.healthCheckSettings.Timeout)
	sleepTicker := clock.NewRealTicker(s.healthCheckSettings.WaitInterval)

	defer timeoutTimer.Stop()
	defer sleepTicker.Stop()

	for {
		sleepTicker.Stop()
		result = s.healthcheck()

		if result.Err() != nil {
			s.logger.Warn("errors during health checks in stage %d: %s", s.index, result.Err())
		}

		if result.IsHealthy() {
			return nil
		}

		for _, unhealthy := range result.GetUnhealthy() {
			s.logger.Info("waiting for module %s in stage %d to get healthy", unhealthy.Name, s.index)
		}

		sleepTicker.Reset(s.healthCheckSettings.WaitInterval)

		select {
		case <-timeoutTimer.Chan():
			unhealthyModules := result.GetUnhealthyNames()

			return fmt.Errorf("stage %d was not able to get healthy in %s due to: %s", s.index, s.healthCheckSettings.Timeout, strings.Join(unhealthyModules, ", "))
		case <-s.ctx.Done():
			return nil
		case <-sleepTicker.Chan():
		}
	}
}

func (s *stage) stopWait() {
	s.cfn.Kill(ErrKernelStopping)
	s.err = s.cfn.Wait()

	if s.err != nil && !errors.Is(s.err, ErrKernelStopping) {
		s.logger.Error("error during the execution of stage %d: %w", s.index, s.err)
	}

	s.terminated.Signal()
}

func (s *stage) len() int {
	return s.modules.len()
}
