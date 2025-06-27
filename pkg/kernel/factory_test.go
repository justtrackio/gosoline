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

	s.config = cfgMocks.NewConfig(s.T())
	s.config.EXPECT().UnmarshalKey("kernel", mock.AnythingOfType("*kernel.Settings")).
		Run(func(key string, val any, _ ...cfg.UnmarshalDefaults) {
			settings := val.(*kernel.Settings)
			settings.KillTimeout = time.Second
			settings.HealthCheck.Timeout = time.Second
			settings.HealthCheck.WaitInterval = time.Second
		})

	s.logger = logMocks.NewLogger(s.T())
	s.logger.EXPECT().WithChannel(mock.AnythingOfType("string")).Return(s.logger)
}

func (s *FactoryTestSuite) TestNoModules() {
	_, err := kernel.BuildFactory(s.ctx, s.config, s.logger, []kernel.Option{})
	s.EqualError(err, "can not build kernel factory: no modules to run")
}

func (s *FactoryTestSuite) TestNoForegroundModules() {
	module := kernelMocks.NewFullModule(s.T())
	module.EXPECT().IsEssential().Return(false).Once()
	module.EXPECT().IsBackground().Return(true).Once()
	module.EXPECT().GetStage().Return(kernel.StageApplication).Once()

	_, err := kernel.BuildFactory(s.ctx, s.config, s.logger, []kernel.Option{
		kernel.WithModuleFactory("background", func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
			return module, nil
		}),
	})
	s.EqualError(err, "can not build kernel factory: no foreground modules to run")
}

func (s *FactoryTestSuite) TestModuleMultiFactoryError() {
	factoryErr := fmt.Errorf("error in module factory")

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
