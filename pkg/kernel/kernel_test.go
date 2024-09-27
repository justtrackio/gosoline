package kernel_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/appctx"
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
	"github.com/stretchr/testify/suite"
	"golang.org/x/sys/unix"
)

type FunctionModule func(ctx context.Context) error

func (m FunctionModule) Run(ctx context.Context) error {
	return m(ctx)
}

func TestKernelTestSuite(t *testing.T) {
	suite.Run(t, new(KernelTestSuite))
}

type KernelTestSuite struct {
	suite.Suite

	ctx    context.Context
	config *cfgMocks.Config
	logger *logMocks.Logger
	module *kernelMocks.FullModule
}

func (s *KernelTestSuite) SetupTest() {
	s.ctx = appctx.WithContainer(context.Background())

	s.config = new(cfgMocks.Config)
	s.config.On("AllSettings").Return(map[string]interface{}{})
	s.config.On("UnmarshalKey", "kernel", mock.AnythingOfType("*kernel.Settings")).Run(func(args mock.Arguments) {
		settings := args[1].(*kernel.Settings)
		settings.KillTimeout = time.Second
		settings.HealthCheck.Timeout = time.Second
		settings.HealthCheck.WaitInterval = time.Second
	})

	s.logger = new(logMocks.Logger)
	s.logger.On("WithChannel", mock.Anything).Return(s.logger)
	s.logger.On("WithFields", mock.Anything).Return(s.logger)
	s.logger.On("Info", mock.Anything)
	s.logger.On("Info", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	s.logger.On("Debug", mock.Anything, mock.Anything)

	s.module = new(kernelMocks.FullModule)
	s.module.On("IsHealthy", mock.AnythingOfType("*context.cancelCtx")).Return(true, nil)
	s.module.On("IsEssential").Return(false)
	s.module.On("IsBackground").Return(false)
}

func (s *KernelTestSuite) TestHangingModule() {
	timeout(s.T(), time.Second*3, func(t *testing.T) {
		logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))

		options := []kernel.Option{
			s.mockExitHandler(kernel.ExitCodeErr),
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

		k, err := kernel.BuildKernel(s.ctx, s.config, logger, options)
		assert.NoError(t, err)

		k.Run()
	})
}

func timeout(t *testing.T, d time.Duration, f func(t *testing.T)) {
	done := make(chan struct{})
	errChan := make(chan error)
	cfn := coffin.New()
	cfn.Go(func() error {
		cfn.Go(func() error {
			defer close(done)
			f(t)

			return nil
		})
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

		return nil
	})

	if err := <-errChan; err != nil {
		assert.FailNow(t, err.Error())
	}

	assert.NoError(t, cfn.Wait())
}

func (s *KernelTestSuite) TestRunSuccess() {
	s.module.On("GetStage").Return(kernel.StageApplication)
	s.module.On("Run", mock.Anything).Return(nil)

	s.NotPanics(func() {
		k, err := kernel.BuildKernel(s.ctx, s.config, s.logger, []kernel.Option{
			kernel.WithModuleFactory("module", func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
				return s.module, nil
			}),
			kernel.WithKillTimeout(time.Second),
			s.mockExitHandler(kernel.ExitCodeOk),
		})
		s.NoError(err)

		k.Run()
	})

	s.module.AssertCalled(s.T(), "Run", mock.Anything)
}

func (s *KernelTestSuite) TestRunFailure() {
	s.logger.On("Error", "error during the execution of stage %d: %w", kernel.StageApplication, mock.Anything)

	s.module.On("GetStage").Return(kernel.StageApplication)
	s.module.On("Run", mock.Anything).Run(func(args mock.Arguments) {
		panic("panic")
	})

	s.NotPanics(func() {
		k, err := kernel.BuildKernel(s.ctx, s.config, s.logger, []kernel.Option{
			kernel.WithModuleFactory("module", func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
				return s.module, nil
			}),
			kernel.WithKillTimeout(time.Second),
			s.mockExitHandler(kernel.ExitCodeErr),
		})
		s.NoError(err)

		k.Run()
	})

	s.module.AssertCalled(s.T(), "Run", mock.Anything)
}

func (s *KernelTestSuite) TestStop() {
	var err error
	var k kernel.Kernel

	s.module.On("IsEssential").Return(false)
	s.module.On("IsBackground").Return(false)
	s.module.On("GetStage").Return(kernel.StageApplication)
	s.module.On("Run", mock.Anything).Run(func(args mock.Arguments) {
		ctx := args.Get(0).(context.Context)

		k.Stop("test done")
		<-ctx.Done()
	}).Return(nil)

	k, err = kernel.BuildKernel(s.ctx, s.config, s.logger, []kernel.Option{
		kernel.WithModuleFactory("module", func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
			return s.module, nil
		}),
		kernel.WithKillTimeout(time.Second),
		s.mockExitHandler(kernel.ExitCodeOk),
	})
	s.NoError(err)

	k.Run()

	s.module.AssertCalled(s.T(), "Run", mock.Anything)
}

func (s *KernelTestSuite) TestRunningType() {
	mf := new(kernelMocks.Module)
	// type = foreground & stage = application are the defaults for a module
	mf.On("Run", mock.Anything).Run(func(args mock.Arguments) {}).Return(nil)

	mb := new(kernelMocks.FullModule)
	mb.On("IsHealthy", mock.AnythingOfType("*context.cancelCtx")).Return(true, nil)
	mb.On("IsEssential").Return(false)
	mb.On("IsBackground").Return(true)
	mb.On("GetStage").Return(kernel.StageApplication)
	mb.On("Run", mock.Anything).Run(func(args mock.Arguments) {
		ctx := args.Get(0).(context.Context)

		<-ctx.Done()
	}).Return(nil)

	k, err := kernel.BuildKernel(s.ctx, s.config, s.logger, []kernel.Option{
		kernel.WithModuleFactory("foreground", func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
			return mf, nil
		}),
		kernel.WithModuleFactory("background", func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
			return mb, nil
		}),
		kernel.WithKillTimeout(time.Second),
		s.mockExitHandler(kernel.ExitCodeOk),
	})
	s.NoError(err)

	k.Run()

	mf.AssertExpectations(s.T())
	mb.AssertExpectations(s.T())
}

func (s *KernelTestSuite) TestMultipleStages() {
	options := []kernel.Option{
		kernel.WithKillTimeout(time.Second),
		s.mockExitHandler(kernel.ExitCodeOk),
	}

	var allMocks []*kernelMocks.FullModule
	var stageStatus []int

	maxStage := 5
	wg := &sync.WaitGroup{}
	wg.Add(maxStage)

	for stage := 0; stage < maxStage; stage++ {
		thisStage := stage

		m := new(kernelMocks.FullModule)
		m.On("IsHealthy", mock.AnythingOfType("*context.cancelCtx")).Return(true, nil)
		m.On("IsEssential").Return(true)
		m.On("IsBackground").Return(false)
		m.On("GetStage").Return(thisStage)
		m.On("Run", mock.Anything).Run(func(args mock.Arguments) {
			ctx := args.Get(0).(context.Context)

			stageStatus[thisStage] = 1

			wg.Done()
			wg.Wait()
			<-ctx.Done()

			s.logger.Info("stage %d: ctx done", thisStage)

			for i := 0; i <= thisStage; i++ {
				s.GreaterOrEqual(stageStatus[i], 1, fmt.Sprintf("stage %d: expected stage %d to be at least running", thisStage, i))
			}
			for i := thisStage + 1; i < maxStage; i++ {
				s.Equal(2, stageStatus[i], fmt.Sprintf("stage %d: expected stage %d to be done", thisStage, i))
			}

			stageStatus[thisStage] = 2
		}).Return(nil)

		allMocks = append(allMocks, m)
		stageStatus = append(stageStatus, 0)

		options = append(options, kernel.WithModuleFactory("m", func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
			return m, nil
		}))
	}

	k, err := kernel.BuildKernel(s.ctx, s.config, s.logger, options)
	s.NoError(err)

	go func() {
		time.Sleep(time.Millisecond * 300)
		k.Stop("we are done testing")
	}()
	k.Run()

	for _, m := range allMocks {
		m.AssertExpectations(s.T())
	}
}

func (s *KernelTestSuite) TestForcedExit() {
	s.logger.On("Error", mock.Anything, mock.Anything)

	mayStop := conc.NewSignalOnce()
	appStopped := conc.NewSignalOnce()

	app := new(kernelMocks.FullModule)
	app.On("IsHealthy", mock.AnythingOfType("*context.cancelCtx")).Return(true, nil)
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
		s.True(appStopped.Signaled())
	}).Return(nil)

	k, err := kernel.BuildKernel(s.ctx, s.config, s.logger, []kernel.Option{
		kernel.WithModuleFactory("m", func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
			return m, nil
		}, kernel.ModuleStage(kernel.StageService), kernel.ModuleType(kernel.TypeForeground)),

		kernel.WithModuleFactory("app", func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
			return app, nil
		}),

		kernel.WithKillTimeout(200 * time.Millisecond),
		kernel.WithExitHandler(func(code int) {
			s.Equal(kernel.ExitCodeForced, code)
			mayStop.Signal()
		}),
	})
	s.NoError(err)

	go func() {
		time.Sleep(time.Millisecond * 300)
		k.Stop("we are done testing")
	}()

	k.Run()

	app.AssertExpectations(s.T())
	m.AssertExpectations(s.T())
	s.True(mayStop.Signaled())
}

func (s *KernelTestSuite) TestStageStopped() {
	s.logger.On("Errorf", mock.Anything, mock.Anything)

	success := false
	appStopped := conc.NewSignalOnce()

	app := new(kernelMocks.FullModule)
	app.On("IsHealthy", mock.AnythingOfType("*context.cancelCtx")).Return(true, nil)
	app.On("IsEssential").Return(false)
	app.On("IsBackground").Return(false)
	app.On("GetStage").Return(kernel.StageApplication)
	app.On("Run", mock.Anything).Run(func(args mock.Arguments) {
		ctx := args.Get(0).(context.Context)

		ticker := time.NewTicker(time.Millisecond * 300)
		defer ticker.Stop()

		select {
		case <-ctx.Done():
			s.T().Fatal("kernel stopped before 300ms")
		case <-ticker.C:
			success = true
		}

		appStopped.Signal()
	}).Return(nil)

	m := new(kernelMocks.FullModule)
	m.On("IsHealthy", mock.AnythingOfType("*context.cancelCtx")).Return(true, nil)
	m.On("IsEssential").Return(false)
	m.On("IsBackground").Return(true)
	m.On("GetStage").Return(777)
	m.On("Run", mock.Anything).Return(nil)

	k, err := kernel.BuildKernel(s.ctx, s.config, s.logger, []kernel.Option{
		kernel.WithModuleFactory("m", func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
			return m, nil
		}),

		kernel.WithModuleFactory("app", func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
			return app, nil
		}),

		kernel.WithKillTimeout(200 * time.Millisecond),
		s.mockExitHandler(kernel.ExitCodeOk),
	})
	s.NoError(err)

	k.Run()

	s.True(success)

	app.AssertExpectations(s.T())
	m.AssertExpectations(s.T())
}

func (s *KernelTestSuite) Test_RunRealModule() {
	// test that we can run the kernel multiple times
	// if this does not work, the next test does not make sense
	for i := 0; i < 10; i++ {
		s.T().Run(fmt.Sprintf("fake iteration %d", i), func(t *testing.T) {
			k, err := kernel.BuildKernel(s.ctx, s.config, s.logger, []kernel.Option{
				s.mockExitHandler(kernel.ExitCodeOk),
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
		s.T().Run(fmt.Sprintf("real iteration %d", i), func(t *testing.T) {
			k, err := kernel.BuildKernel(s.ctx, s.config, s.logger, []kernel.Option{
				s.mockExitHandler(kernel.ExitCodeOk),
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

func (s *KernelTestSuite) TestModuleFastShutdown() {
	var err error
	var k kernel.Kernel

	options := []kernel.Option{s.mockExitHandler(kernel.ExitCodeOk)}

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

	k, err = kernel.BuildKernel(s.ctx, s.config, s.logger, options)
	s.NoError(err)

	k.Run()
}

func (s *KernelTestSuite) mockExitHandler(expectedCode int) kernel.Option {
	return kernel.WithExitHandler(func(actualCode int) {
		s.Equal(expectedCode, actualCode, "exit code does not match")
	})
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
