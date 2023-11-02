package kernel_test

import (
	"context"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	cfgMocks "github.com/justtrackio/gosoline/pkg/cfg/mocks"
	"github.com/justtrackio/gosoline/pkg/kernel"
	kernelMocks "github.com/justtrackio/gosoline/pkg/kernel/mocks"
	"github.com/justtrackio/gosoline/pkg/log"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

func TestMiddleWareTestSuite(t *testing.T) {
	suite.Run(t, new(MiddleWareTestSuite))
}

type MiddleWareTestSuite struct {
	suite.Suite

	ctx    context.Context
	config *cfgMocks.Config
	logger *logMocks.Logger
	module *kernelMocks.FullModule
}

func (s *MiddleWareTestSuite) SetupTest() {
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
	s.module.On("IsEssential").Return(false)
	s.module.On("IsBackground").Return(false)
}

func (s *MiddleWareTestSuite) TestMiddleware() {
	callstack := make([]string, 0)

	s.module.On("IsHealthy", mock.AnythingOfType("*context.cancelCtx")).Return(true, nil)
	s.module.On("GetStage").Return(kernel.StageApplication)
	s.module.On("Run", mock.Anything).Run(func(args mock.Arguments) {
		callstack = append(callstack, "module")
	}).Return(nil)

	k, err := kernel.BuildKernel(s.ctx, s.config, s.logger, []kernel.Option{
		kernel.WithMiddlewareFactory(kernel.BuildSimpleMiddleware(func(next kernel.MiddlewareHandler) {
			callstack = append(callstack, "mid1 start")
			next()
			callstack = append(callstack, "mid1 end")
		}), kernel.PositionEnd),

		kernel.WithMiddlewareFactory(kernel.BuildSimpleMiddleware(func(next kernel.MiddlewareHandler) {
			callstack = append(callstack, "mid2 start")
			next()
			callstack = append(callstack, "mid2 end")
		}), kernel.PositionEnd),

		kernel.WithMiddlewareFactory(kernel.BuildSimpleMiddleware(func(next kernel.MiddlewareHandler) {
			callstack = append(callstack, "mid3 start")
			next()
			callstack = append(callstack, "mid3 end")
		}), kernel.PositionBeginning),

		kernel.WithModuleFactory("module", func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
			return s.module, nil
		}),

		kernel.WithKillTimeout(time.Second),
		s.mockExitHandler(kernel.ExitCodeOk),
	})

	s.NoError(err)
	k.Run()

	expectedCallstack := []string{
		"mid3 start",
		"mid1 start",
		"mid2 start",
		"module",
		"mid2 end",
		"mid1 end",
		"mid3 end",
	}

	s.Equal(expectedCallstack, callstack)
}

func (s *MiddleWareTestSuite) mockExitHandler(expectedCode int) kernel.Option {
	return kernel.WithExitHandler(func(actualCode int) {
		s.Equal(expectedCode, actualCode, "exit code does not match")
	})
}
