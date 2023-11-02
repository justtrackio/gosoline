package kernel_test

import (
	"context"
	"fmt"
	"strings"
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

func TestFactoryTestSuite(t *testing.T) {
	suite.Run(t, new(FactoryTestSuite))
}

type FactoryTestSuite struct {
	suite.Suite

	ctx    context.Context
	config *cfgMocks.Config
	logger *logMocks.Logger
}

func (s *FactoryTestSuite) SetupTest() {
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
}

func (s *FactoryTestSuite) TestNoModules() {
	_, err := kernel.BuildFactory(s.ctx, s.config, s.logger, []kernel.Option{})
	s.EqualError(err, "can not build kernel factory: no modules to run")
}

func (s *FactoryTestSuite) TestNoForegroundModules() {
	module := new(kernelMocks.FullModule)
	module.On("IsEssential").Return(false)
	module.On("IsBackground").Return(true)
	module.On("GetStage").Return(kernel.StageApplication)
	module.On("Run", mock.Anything).Return(nil)

	_, err := kernel.BuildFactory(s.ctx, s.config, s.logger, []kernel.Option{
		kernel.WithModuleFactory("background", func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
			return module, nil
		}),
	})
	s.EqualError(err, "can not build kernel factory: no foreground modules to run")
}

func (s *FactoryTestSuite) TestModuleMultiFactoryError() {
	factoryErr := fmt.Errorf("error in module factory")
	s.logger.On("Error", "error building additional modules by multiFactories: %w", factoryErr)

	_, err := kernel.BuildFactory(s.ctx, s.config, s.logger, []kernel.Option{
		kernel.WithModuleMultiFactory(func(context.Context, cfg.Config, log.Logger) (map[string]kernel.ModuleFactory, error) {
			return nil, factoryErr
		}),
	})

	s.EqualError(err, "can not build kernel factory: error in module factory")
}

func (s *FactoryTestSuite) TestModuleMultiFactoryPanic() {
	_, err := kernel.BuildFactory(s.ctx, s.config, s.logger, []kernel.Option{
		kernel.WithModuleMultiFactory(func(context.Context, cfg.Config, log.Logger) (map[string]kernel.ModuleFactory, error) {
			panic("panic in module multi factory")
		}),
	})

	s.True(strings.Contains(err.Error(), "can not build kernel factory: panic in module multi factory"))
}

func (s *FactoryTestSuite) TestModuleFactoryPanic() {
	_, err := kernel.BuildFactory(s.ctx, s.config, s.logger, []kernel.Option{
		kernel.WithModuleFactory("module", func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
			panic("panic in module factory")
		}),
	})

	s.True(strings.Contains(err.Error(), "can not build kernel factory: panic in module factory"))
}
