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
	"github.com/justtrackio/gosoline/pkg/test/matcher"
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
	s.ctx = appctx.WithContainer(s.T().Context())

	s.config = cfgMocks.NewConfig(s.T())
	s.config.EXPECT().UnmarshalKey("kernel", mock.AnythingOfType("*kernel.Settings")).Run(
		func(key string, val any, _ ...cfg.UnmarshalDefaults) {
			settings := val.(*kernel.Settings)
			settings.KillTimeout = time.Second
			settings.HealthCheck.Timeout = time.Second
			settings.HealthCheck.WaitInterval = time.Second
		}).
		Return(nil)

	s.logger = logMocks.NewLogger(s.T())
	s.logger.EXPECT().WithChannel(mock.AnythingOfType("string")).Return(s.logger)
	s.logger.EXPECT().Info(matcher.Context, mock.Anything)
	s.logger.EXPECT().Info(matcher.Context, mock.Anything, mock.Anything, mock.Anything, mock.Anything)

	s.module = kernelMocks.NewFullModule(s.T())
	s.module.EXPECT().IsEssential().Return(false).Once()
	s.module.EXPECT().IsBackground().Return(false).Once()
	s.module.EXPECT().IsHealthy(matcher.Context).Return(true, nil).Once()
	s.module.EXPECT().GetStage().Return(kernel.StageApplication).Once()
}

func (s *MiddleWareTestSuite) TestMiddleware() {
	callstack := make([]string, 0)

	s.module.EXPECT().Run(matcher.Context).Run(func(ctx context.Context) {
		callstack = append(callstack, "module")
	}).Return(nil).Once()

	k, err := kernel.BuildKernel(s.ctx, s.config, s.logger, []kernel.Option{
		kernel.WithMiddlewareFactory(kernel.BuildSimpleMiddleware(func(next kernel.MiddlewareHandler) {
			callstack = append(callstack, "mid1 start")
			next(s.ctx)
			callstack = append(callstack, "mid1 end")
		}), kernel.PositionEnd),

		kernel.WithMiddlewareFactory(kernel.BuildSimpleMiddleware(func(next kernel.MiddlewareHandler) {
			callstack = append(callstack, "mid2 start")
			next(s.ctx)
			callstack = append(callstack, "mid2 end")
		}), kernel.PositionEnd),

		kernel.WithMiddlewareFactory(kernel.BuildSimpleMiddleware(func(next kernel.MiddlewareHandler) {
			callstack = append(callstack, "mid3 start")
			next(s.ctx)
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
