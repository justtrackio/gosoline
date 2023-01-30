package kernel_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	cfgMocks "github.com/justtrackio/gosoline/pkg/cfg/mocks"
	"github.com/justtrackio/gosoline/pkg/coffin"
	"github.com/justtrackio/gosoline/pkg/conc"
	"github.com/justtrackio/gosoline/pkg/kernel"
	kernelMocks "github.com/justtrackio/gosoline/pkg/kernel/mocks"
	"github.com/justtrackio/gosoline/pkg/log"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/sys/unix"
)

type FunctionModule func(ctx context.Context) error

func (m FunctionModule) Run(ctx context.Context) error {
	return m(ctx)
}

func TestKernelHangingModule(t *testing.T) {
	timeout(t, time.Second*3, func(t *testing.T) {
		config, _, _ := createMocks()
		logger := logMocks.NewLoggerMockedAll()

		options := []kernel.Option{
			mockExitHandler(t, kernel.ExitCodeErr),
		}

		options = append(options, kernel.WithModuleFactory("normal module", func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
			return FunctionModule(func(ctx context.Context) error {
				<-ctx.Done()

				return nil
			}), nil
		}, kernel.ModuleStage(kernel.StageApplication), kernel.ModuleType(kernel.TypeForeground)))

		serviceChannel := make(chan int)
		options = append(options, kernel.WithModuleFactory("service module", func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
			return FunctionModule(func(ctx context.Context) error {
				processed := 0

				for {
					select {
					case <-ctx.Done():
						return nil
					case <-serviceChannel:
						processed++
						if processed > 3 {
							return fmt.Errorf("random fail")
						}
					}
				}
			}), nil
		}, kernel.ModuleStage(kernel.StageService), kernel.ModuleType(kernel.TypeBackground)))

		options = append(options, kernel.WithModuleFactory("hanging module", func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
			return FunctionModule(func(ctx context.Context) error {
				n := 0
				for {
					select {
					case <-ctx.Done():
						return nil
					case serviceChannel <- n:
						n++
					}
				}
			}), nil
		}, kernel.ModuleStage(kernel.StageService), kernel.ModuleType(kernel.TypeForeground)))

		k, err := kernel.BuildKernel(context.Background(), config, logger, options)
		assert.NoError(t, err)

		k.Run()
	})
}

func timeout(t *testing.T, d time.Duration, f func(t *testing.T)) {
	done := make(chan struct{})
	cfn := coffin.New()
	cfn.Go(func() error {
		defer close(done)
		f(t)

		return nil
	})
	errChan := make(chan error)
	cfn.Go(func() error {
		timer := time.NewTimer(d)
		defer timer.Stop()
		defer close(errChan)

		select {
		case <-timer.C:
			errChan <- fmt.Errorf("test timed out after %v", d)
		case <-done:
		}

		return nil
	})

	if err := <-errChan; err != nil {
		assert.FailNow(t, err.Error())
	}

	assert.NoError(t, cfn.Wait())
}

func createMocks() (*cfgMocks.Config, *logMocks.Logger, *kernelMocks.FullModule) {
	config := new(cfgMocks.Config)
	config.On("AllSettings").Return(map[string]interface{}{})
	config.On("UnmarshalKey", "kernel", mock.AnythingOfType("*kernel.Settings")).Return(map[string]interface{}{})

	logger := new(logMocks.Logger)
	logger.On("WithChannel", mock.Anything).Return(logger)
	logger.On("WithFields", mock.Anything).Return(logger)
	logger.On("Info", mock.Anything)
	logger.On("Info", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	logger.On("Debug", mock.Anything, mock.Anything)

	module := new(kernelMocks.FullModule)
	module.On("IsEssential").Return(false)
	module.On("IsBackground").Return(false)

	return config, logger, module
}

func mockExitHandler(t *testing.T, expectedCode int) kernel.Option {
	return kernel.WithExitHandler(func(actualCode int) {
		assert.Equal(t, expectedCode, actualCode, "exit code does not match")
	})
}

func TestKernelRunSuccess(t *testing.T) {
	config, logger, module := createMocks()

	module.On("GetStage").Return(kernel.StageApplication)
	module.On("Run", mock.Anything).Return(nil)

	assert.NotPanics(t, func() {
		k, err := kernel.BuildKernel(context.Background(), config, logger, []kernel.Option{
			kernel.WithModuleFactory("module", func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
				return module, nil
			}),
			kernel.WithKillTimeout(time.Second),
			mockExitHandler(t, kernel.ExitCodeOk),
		})
		assert.NoError(t, err)

		k.Run()
	})

	module.AssertCalled(t, "Run", mock.Anything)
}

func TestKernelRunFailure(t *testing.T) {
	config, logger, module := createMocks()

	logger.On("Error", "error during the execution of stage %d: %w", kernel.StageApplication, mock.Anything)

	module.On("GetStage").Return(kernel.StageApplication)
	module.On("Run", mock.Anything).Run(func(args mock.Arguments) {
		panic("panic")
	})

	assert.NotPanics(t, func() {
		k, err := kernel.BuildKernel(context.Background(), config, logger, []kernel.Option{
			kernel.WithModuleFactory("module", func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
				return module, nil
			}),
			kernel.WithKillTimeout(time.Second),
			mockExitHandler(t, kernel.ExitCodeErr),
		})
		assert.NoError(t, err)

		k.Run()
	})

	module.AssertCalled(t, "Run", mock.Anything)
}

func TestKernelStop(t *testing.T) {
	config, logger, module := createMocks()

	var err error
	var k kernel.Kernel

	module.On("IsEssential").Return(false)
	module.On("IsBackground").Return(false)
	module.On("GetStage").Return(kernel.StageApplication)
	module.On("Run", mock.Anything).Run(func(args mock.Arguments) {
		ctx := args.Get(0).(context.Context)

		k.Stop("test done")
		<-ctx.Done()
	}).Return(nil)

	k, err = kernel.BuildKernel(context.Background(), config, logger, []kernel.Option{
		kernel.WithModuleFactory("module", func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
			return module, nil
		}),
		kernel.WithKillTimeout(time.Second),
		mockExitHandler(t, kernel.ExitCodeOk),
	})
	assert.NoError(t, err)

	k.Run()

	module.AssertCalled(t, "Run", mock.Anything)
}

func TestKernelRunningType(t *testing.T) {
	config, logger, _ := createMocks()

	mf := new(kernelMocks.Module)
	// type = foreground & stage = application are the defaults for a module
	mf.On("Run", mock.Anything).Run(func(args mock.Arguments) {}).Return(nil)

	mb := new(kernelMocks.FullModule)
	mb.On("IsEssential").Return(false)
	mb.On("IsBackground").Return(true)
	mb.On("GetStage").Return(kernel.StageApplication)
	mb.On("Run", mock.Anything).Run(func(args mock.Arguments) {
		ctx := args.Get(0).(context.Context)

		<-ctx.Done()
	}).Return(nil)

	k, err := kernel.BuildKernel(context.Background(), config, logger, []kernel.Option{
		kernel.WithModuleFactory("foreground", func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
			return mf, nil
		}),
		kernel.WithModuleFactory("background", func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
			return mb, nil
		}),
		kernel.WithKillTimeout(time.Second),
		mockExitHandler(t, kernel.ExitCodeOk),
	})
	assert.NoError(t, err)

	k.Run()

	mf.AssertExpectations(t)
	mb.AssertExpectations(t)
}

func TestKernelMultipleStages(t *testing.T) {
	config, logger, _ := createMocks()

	options := []kernel.Option{
		kernel.WithKillTimeout(time.Second),
		mockExitHandler(t, kernel.ExitCodeOk),
	}

	var allMocks []*kernelMocks.FullModule
	var stageStatus []int

	maxStage := 5
	wg := &sync.WaitGroup{}
	wg.Add(maxStage)

	for stage := 0; stage < maxStage; stage++ {
		thisStage := stage

		m := new(kernelMocks.FullModule)
		m.On("IsEssential").Return(true)
		m.On("IsBackground").Return(false)
		m.On("GetStage").Return(thisStage)
		m.On("Run", mock.Anything).Run(func(args mock.Arguments) {
			ctx := args.Get(0).(context.Context)

			stageStatus[thisStage] = 1

			wg.Done()
			wg.Wait()
			<-ctx.Done()

			logger.Info("stage %d: ctx done", thisStage)

			for i := 0; i <= thisStage; i++ {
				assert.GreaterOrEqual(t, stageStatus[i], 1, fmt.Sprintf("stage %d: expected stage %d to be at least running", thisStage, i))
			}
			for i := thisStage + 1; i < maxStage; i++ {
				assert.Equal(t, 2, stageStatus[i], fmt.Sprintf("stage %d: expected stage %d to be done", thisStage, i))
			}

			stageStatus[thisStage] = 2
		}).Return(nil)

		allMocks = append(allMocks, m)
		stageStatus = append(stageStatus, 0)

		options = append(options, kernel.WithModuleFactory("m", func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
			return m, nil
		}))
	}

	k, err := kernel.BuildKernel(context.Background(), config, logger, options)
	assert.NoError(t, err)

	go func() {
		time.Sleep(time.Millisecond * 300)
		k.Stop("we are done testing")
	}()
	k.Run()

	for _, m := range allMocks {
		m.AssertExpectations(t)
	}
}

func TestKernelForcedExit(t *testing.T) {
	config, logger, _ := createMocks()
	logger.On("Error", mock.Anything, mock.Anything)

	mayStop := conc.NewSignalOnce()
	appStopped := conc.NewSignalOnce()

	app := new(kernelMocks.FullModule)
	app.On("IsEssential").Return(false)
	app.On("IsBackground").Return(true)
	app.On("GetStage").Return(kernel.StageApplication)
	app.On("Run", mock.Anything).Run(func(args mock.Arguments) {
		ctx := args.Get(0).(context.Context)

		<-ctx.Done()
		appStopped.Signal()
	}).Return(nil)

	m := new(kernelMocks.Module)
	m.On("Run", mock.Anything).Run(func(args mock.Arguments) {
		<-mayStop.Channel()
		assert.True(t, appStopped.Signaled())
	}).Return(nil)

	k, err := kernel.BuildKernel(context.Background(), config, logger, []kernel.Option{
		kernel.WithModuleFactory("m", func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
			return m, nil
		}, kernel.ModuleStage(kernel.StageService), kernel.ModuleType(kernel.TypeForeground)),

		kernel.WithModuleFactory("app", func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
			return app, nil
		}),

		kernel.WithKillTimeout(200 * time.Millisecond),
		kernel.WithExitHandler(func(code int) {
			assert.Equal(t, kernel.ExitCodeForced, code)
			mayStop.Signal()
		}),
	})
	assert.NoError(t, err)

	go func() {
		time.Sleep(time.Millisecond * 300)
		k.Stop("we are done testing")
	}()

	k.Run()

	app.AssertExpectations(t)
	m.AssertExpectations(t)
	assert.True(t, mayStop.Signaled())
}

func TestKernelStageStopped(t *testing.T) {
	config, logger, _ := createMocks()
	logger.On("Errorf", mock.Anything, mock.Anything)

	success := false
	appStopped := conc.NewSignalOnce()

	app := new(kernelMocks.FullModule)
	app.On("IsEssential").Return(false)
	app.On("IsBackground").Return(false)
	app.On("GetStage").Return(kernel.StageApplication)
	app.On("Run", mock.Anything).Run(func(args mock.Arguments) {
		ctx := args.Get(0).(context.Context)

		ticker := time.NewTicker(time.Millisecond * 300)
		defer ticker.Stop()

		select {
		case <-ctx.Done():
			t.Fatal("kernel stopped before 300ms")
		case <-ticker.C:
			success = true
		}

		appStopped.Signal()
	}).Return(nil)

	m := new(kernelMocks.FullModule)
	m.On("IsEssential").Return(false)
	m.On("IsBackground").Return(true)
	m.On("GetStage").Return(777)
	m.On("Run", mock.Anything).Return(nil)

	k, err := kernel.BuildKernel(context.Background(), config, logger, []kernel.Option{
		kernel.WithModuleFactory("m", func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
			return m, nil
		}),

		kernel.WithModuleFactory("app", func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
			return app, nil
		}),

		kernel.WithKillTimeout(200 * time.Millisecond),
		mockExitHandler(t, kernel.ExitCodeOk),
	})
	assert.NoError(t, err)

	k.Run()

	assert.True(t, success)

	app.AssertExpectations(t)
	m.AssertExpectations(t)
}

type fakeModule struct{}

func (m *fakeModule) Run(_ context.Context) error {
	return nil
}

type realModule struct {
	t *testing.T
}

func (m *realModule) Run(ctx context.Context) error {
	cfn, cfnCtx := coffin.WithContext(ctx)

	cfn.GoWithContext(cfnCtx, func(ctx context.Context) error {
		ticker := time.NewTicker(time.Millisecond * 2)
		defer ticker.Stop()

		counter := 0

		for {
			select {
			case <-ticker.C:
				counter++
				if counter == 3 {
					err := unix.Kill(unix.Getpid(), unix.SIGTERM)
					assert.NoError(m.t, err)
				}
			case <-ctx.Done():
				return nil
			}
		}
	})

	err := cfn.Wait()
	if !errors.Is(err, context.Canceled) {
		assert.NoError(m.t, err)
	}
	return err
}

func TestKernel_RunRealModule(t *testing.T) {
	// test that we can run the kernel multiple times
	// if this does not work, the next test does not make sense
	for i := 0; i < 10; i++ {
		t.Run(fmt.Sprintf("fake iteration %d", i), func(t *testing.T) {
			config, logger, _ := createMocks()

			k, err := kernel.BuildKernel(context.Background(), config, logger, []kernel.Option{
				mockExitHandler(t, kernel.ExitCodeOk),
				kernel.WithModuleFactory("main", func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
					return &fakeModule{}, nil
				}),
			})
			assert.NoError(t, err)

			k.Run()
		})
	}
	// test for a race condition on kernel shutdown
	// in the past, this would panic in a close on closed channel in the tomb module
	for i := 0; i < 10; i++ {
		t.Run(fmt.Sprintf("real iteration %d", i), func(t *testing.T) {
			config, logger, _ := createMocks()

			k, err := kernel.BuildKernel(context.Background(), config, logger, []kernel.Option{
				mockExitHandler(t, kernel.ExitCodeOk),
				kernel.WithModuleFactory("main", func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
					return &realModule{
						t: t,
					}, nil
				}),
			})
			assert.NoError(t, err)

			k.Run()
		})
	}
}

type fastExitModule struct {
	kernel.BackgroundModule
}

func (f *fastExitModule) Run(_ context.Context) error {
	return nil
}

type slowExitModule struct {
	fastExitModule
	kernel.ForegroundModule
	stop func()
}

func (s *slowExitModule) Run(_ context.Context) error {
	s.stop()
	return nil
}

func TestModuleFastShutdown(t *testing.T) {
	var err error
	var k kernel.Kernel

	config, logger, _ := createMocks()
	options := []kernel.Option{mockExitHandler(t, kernel.ExitCodeOk)}

	for s := 5; s < 10; s++ {
		options = append(options, kernel.WithModuleFactory("exist-fast", func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
			return &fastExitModule{}, nil
		}, kernel.ModuleStage(s)))

		options = append(options, kernel.WithModuleFactory("exist-slow", func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
			return &slowExitModule{
				stop: func() {
					k.Stop("slowly")
				},
			}, nil
		}, kernel.ModuleStage(s)))
	}

	k, err = kernel.BuildKernel(context.Background(), config, logger, options)
	assert.NoError(t, err)

	k.Run()
}
