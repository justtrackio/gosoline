package kernel

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
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

//go:generate mockery --name Kernel
type Kernel interface {
	Running() <-chan struct{}
	Run()
	Stop(reason string)
}

type kernelOption func(k *kernel)

type kernel struct {
	ctx    context.Context
	config cfg.Config
	logger log.Logger

	middlewares       []Middleware
	stages            stages
	running           chan struct{}
	stopOnce          sync.Once
	foregroundModules int32

	killTimeout time.Duration
	exitCode    int
	exitOnce    sync.Once
	exitHandler ExitHandler
}

func New(ctx context.Context, config cfg.Config, logger log.Logger, middlewares []Middleware, stages map[int]*stage) *kernel {
	k := &kernel{
		config: config,
		logger: logger.WithChannel("kernel"),

		ctx:         ctx,
		middlewares: middlewares,
		stages:      stages,
		running:     make(chan struct{}),

		killTimeout: time.Second * 10,
		exitCode:    ExitCodeErr,
		exitHandler: os.Exit,
	}

	return k
}

// Run will boot and run the modules added to the kernel.
// By default, os.Exit will get called if an error occurs or after the modules have stopped running,
// which means that there will be no return out of this call.
func (k *kernel) Run() {
	defer k.exit()

	k.logger.Info("starting kernel")
	k.foregroundModules = k.stages.countForegroundModules()
	k.debugConfig()

	runHandler := func() {
		sig := make(chan os.Signal, 2)
		signal.Notify(sig, unix.SIGTERM, unix.SIGINT)

		for _, stageIndex := range k.stages.getIndices() {
			k.stages[stageIndex].run(k)
			k.logger.Info("stage %d up and running with %d modules", stageIndex, k.stages[stageIndex].len())
		}

		k.logger.Info("kernel up and running")
		close(k.running)

		select {
		case <-k.waitAllStagesDone().Channel():
			k.Stop("context done")
		case sig := <-sig:
			reason := fmt.Sprintf("signal %s", sig.String())
			k.Stop(reason)
		}

		k.waitStopped()

		hasErr := false
		for _, stage := range k.stages {
			if stage.err != nil && stage.err != ErrKernelStopping {
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

	runHandler()
}

func (k *kernel) Stop(reason string) {
	k.stopOnce.Do(func() {
		go func() {
			k.logger.Info("stopping kernel due to: %s", reason)
			indices := k.stages.getIndices()

			for i := len(indices) - 1; i >= 0; i-- {
				stageIndex := indices[i]
				k.logger.Info("stopping stage %d", stageIndex)
				k.stages[stageIndex].stopWait(stageIndex, k.logger)
				k.logger.Info("stopped stage %d", stageIndex)
			}
		}()
	})
}

func (k *kernel) Running() <-chan struct{} {
	return k.running
}

func (k *kernel) exit() {
	k.exitOnce.Do(func() {
		k.logger.Info("leaving kernel with exit code %d", k.exitCode)
		k.exitHandler(k.exitCode)
	})
}

func (k *kernel) debugConfig() {
	debugErr := cfg.DebugConfig(k.config, k.logger)

	if debugErr != nil {
		k.logger.Error("can not debug config: %w", debugErr)
	}
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

			// we don't need to iterate in order, but doing so is much nicer, so lets do it
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
