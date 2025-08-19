package kernel

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/coffin"
	"github.com/justtrackio/gosoline/pkg/conc"
	"github.com/justtrackio/gosoline/pkg/log"
	"golang.org/x/sys/unix"
)

const (
	ExitCodeOk           = 0
	ExitCodeErr          = 1
	ExitCodeNothingToRun = 10
	ExitCodeNoForeground = 11
	ExitCodeForced       = 12
)

type ExitHandler func(code int)

type Settings struct {
	KillTimeout time.Duration       `cfg:"kill_timeout" default:"10s"`
	HealthCheck HealthCheckSettings `cfg:"health_check"`
}

//go:generate go run github.com/vektra/mockery/v2 --name Kernel
type Kernel interface {
	HealthCheck() HealthCheckResult
	Running() <-chan struct{}
	Run()
	Stop(reason string)
}

type kernelOption func(k *kernel)

type kernel struct {
	ctx    context.Context
	clock  clock.Clock
	logger log.Logger

	middlewareCtx    context.Context
	middlewareCancel func()
	middlewares      []Middleware

	stages            stages
	running           chan struct{}
	stopping          chan struct{}
	stopOnce          sync.Once
	foregroundModules int32

	killTimeout time.Duration
	exitCode    int
	exitOnce    sync.Once
	exitHandler ExitHandler
}

func newKernel(ctx context.Context, config cfg.Config, logger log.Logger) (*kernel, error) {
	settings, err := readSettings(config)
	if err != nil {
		return nil, fmt.Errorf("failed to read kernel settings: %w", err)
	}

	k := &kernel{
		logger: logger.WithChannel("kernel"),
		clock:  clock.NewRealClock(),

		ctx:      ctx,
		running:  make(chan struct{}),
		stopping: make(chan struct{}),

		killTimeout: settings.KillTimeout,
		exitCode:    ExitCodeErr,
		exitHandler: os.Exit,
	}

	_, err = appctx.Provide(ctx, healthCheckerKey, func() (HealthChecker, error) {
		return k.HealthCheck, nil
	})

	return k, err
}

func (k *kernel) init(ctx context.Context, middlewares []Middleware, stages map[int]*stage) {
	k.middlewares = middlewares
	k.middlewareCtx, k.middlewareCancel = context.WithCancel(ctx)
	k.stages = stages
}

// Run will boot and run the modules added to the kernel.
// By default, os.Exit will get called if an error occurs or after the modules have stopped running,
// which means that there will be no return out of this call.
func (k *kernel) Run() {
	defer k.exit()
	defer func() {
		if err := coffin.ResolveRecovery(recover()); err != nil {
			k.logger.WithContext(k.ctx).Error("failed to run kernel: %w", err)
			k.exitCode = ExitCodeErr
		}
	}()

	startedAt := k.clock.Now()
	k.logger.Info("starting kernel")
	k.foregroundModules = k.stages.countForegroundModules()

	sig := make(chan os.Signal, 2)
	signal.Notify(sig, unix.SIGTERM, unix.SIGINT)
	defer func() {
		// stop receiving signals on that channel and close it to avoid leaking a go routine
		signal.Stop(sig)
		close(sig)
	}()

	go func() {
		receivedSignal, ok := <-sig
		if ok {
			reason := fmt.Sprintf("signal %s", receivedSignal.String())
			k.Stop(reason)
		}
	}()

	runHandler := func(ctx context.Context) {
		if err := k.runStages(); err != nil {
			reason := fmt.Sprintf("error during running all stages: %s", err)
			k.Stop(reason)
		}

		took := k.clock.Since(startedAt)
		k.logger.Info("kernel up and running after %s", took)
		close(k.running)

		<-k.waitAllStagesDone().Channel()
		k.Stop("context done")

		k.waitStopped()

		hasErr := false
		for _, stage := range k.stages {
			if stage.err != nil && !errors.Is(stage.err, ErrKernelStopping) {
				hasErr = true
			}
		}

		if !hasErr {
			k.exitCode = ExitCodeOk
		}
	}

	for i := len(k.middlewares) - 1; i >= 0; i-- {
		runHandler = k.middlewares[i](runHandler)
	}

	runHandler(k.middlewareCtx)
}

func (k *kernel) Stop(reason string) {
	k.stop(reason, 0, nil)
}

func (k *kernel) stop(reason string, moduleStage int, moduleErr error) {
	k.stopOnce.Do(func() {
		close(k.stopping)

		go func() {
			k.logger.Info("stopping kernel due to: %s", reason)
			indices := k.stages.getIndices()

			for i := len(indices) - 1; i >= 0; i-- {
				stageIndex := indices[i]
				k.logger.Info("stopping stage %d", stageIndex)

				waitErr := moduleErr
				if stageIndex != moduleStage {
					waitErr = nil
				}
				k.stages[stageIndex].stopWait(waitErr)

				k.logger.Info("stopped stage %d", stageIndex)
			}

			k.middlewareCancel()
		}()
	})
}

func (k *kernel) Running() <-chan struct{} {
	return k.running
}

func (k *kernel) HealthCheck() HealthCheckResult {
	result := make(HealthCheckResult, 0, len(k.stages))

	for _, stageIndex := range k.stages.getIndices() {
		stageResult := k.stages[stageIndex].healthcheck()
		result = append(result, stageResult...)
	}

	slices.SortFunc(result, func(a, b ModuleHealthCheckResult) int {
		return cmp.Compare(a.StageIndex, b.StageIndex)
	})

	if !result.IsHealthy() && k.isRunning() && !k.isStopping() {
		k.reportFailedHealthcheck(result)
	}

	return result
}

func (k *kernel) isRunning() bool {
	select {
	case <-k.running:
		return true
	default:
		return false
	}
}

func (k *kernel) isStopping() bool {
	select {
	case <-k.stopping:
		return true
	default:
		return false
	}
}

func (k *kernel) reportFailedHealthcheck(result HealthCheckResult) {
	unhealthy := result.GetUnhealthyNames()

	buf := make([]byte, 1<<20)
	written := runtime.Stack(buf, true)
	buf = buf[:written]

	k.logger.WithContext(k.ctx).Error("healthcheck failed, unhealthy modules: %s\n%s", unhealthy, string(buf))
}

func (k *kernel) exit() {
	k.exitOnce.Do(func() {
		k.logger.Info("leaving kernel with exit code %d", k.exitCode)
		k.exitHandler(k.exitCode)
	})
}

func (k *kernel) runStages() error {
	for _, stageIndex := range k.stages.getIndices() {
		if err := k.stages[stageIndex].run(k); err != nil {
			return fmt.Errorf("can not run stage %d: %w", stageIndex, err)
		}

		k.logger.Info("stage %d up and running with %d modules", stageIndex, k.stages[stageIndex].len())
	}

	return nil
}

func (k *kernel) runModule(ctx context.Context, name string, ms *moduleState) (moduleErr error) {
	defer k.logger.Info("stopped %s module %s", ms.config.GetType(), name)

	k.logger.Info("running %s module %s in stage %d", ms.config.GetType(), name, ms.config.stage)

	atomic.StoreInt32(&ms.isRunning, 1)

	defer func(ms *moduleState) {
		// recover any crash from the module - if we let the coffin handle this,
		// this is already too late because we might have killed the kernel and
		// swallowed the error
		panicErr := coffin.ResolveRecovery(recover())

		if panicErr != nil {
			ms.err = panicErr
		}

		if ms.err != nil {
			k.logger.Error("error running %s module %s: %w", ms.config.GetType(), name, ms.err)
		}

		atomic.StoreInt32(&ms.isRunning, 0)
		if ms.config.essential {
			k.essentialModuleExited(name)
		} else if !ms.config.background {
			k.foregroundModuleExited()
		}

		// make sure we are returning the correct error to our caller
		moduleErr = ms.err
	}(ms)

	ms.err = ms.module.Run(ctx)

	return ms.err
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
			k.logger.Error("kernel shutdown seems to be blocking.. exiting...: %w", err)

			// we don't need to iterate in order, but doing so is much nicer, so let's do it
			for _, stageIndex := range k.stages.getIndices() {
				s := k.stages[stageIndex]
				for name, ms := range s.modules.modules {
					if atomic.LoadInt32(&ms.isRunning) != 0 {
						k.logger.Info("module in stage %d blocking the shutdown: %s", stageIndex, name)
					}
				}
			}

			k.exitCode = ExitCodeForced
			k.exit()
		case <-done.Channel():
			return
		}
	}()

	// we don't need to iterate in order, we just need to block until everything is done
	for _, stage := range k.stages {
		<-stage.terminated.Channel()
	}
}

func (k *kernel) waitAllStagesDone() conc.SignalOnce {
	done := conc.NewSignalOnce()

	go func() {
		for _, s := range k.stages {
			<-s.ctx.Done()
		}

		done.Signal()
	}()

	return done
}

func readSettings(config cfg.Config) (Settings, error) {
	settings := Settings{}
	if err := config.UnmarshalKey("kernel", &settings); err != nil {
		return Settings{}, fmt.Errorf("failed to unmarshal kernel settings: %w", err)
	}

	return settings, nil
}
