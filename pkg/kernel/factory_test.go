package kernel_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kernel"
	kernelMocks "github.com/justtrackio/gosoline/pkg/kernel/mocks"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestFactoryNoModules(t *testing.T) {
	config, logger, _ := createMocks()

	_, err := kernel.BuildFactory(context.Background(), config, logger, []kernel.Option{})
	assert.EqualError(t, err, "can not build kernel factory: no modules to run")
}

func TestFactoryNoForegroundModules(t *testing.T) {
	config, logger, _ := createMocks()

	module := new(kernelMocks.FullModule)
	module.On("IsEssential").Return(false)
	module.On("IsBackground").Return(true)
	module.On("GetStage").Return(kernel.StageApplication)
	module.On("Run", mock.Anything).Return(nil)

	_, err := kernel.BuildFactory(context.Background(), config, logger, []kernel.Option{
		kernel.WithModuleFactory("background", func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
			return module, nil
		}),
	})
	assert.EqualError(t, err, "can not build kernel factory: no foreground modules to run")
}

func TestFactoryModuleMultiFactoryError(t *testing.T) {
	config, logger, _ := createMocks()
	factoryErr := fmt.Errorf("error in module factory")
	logger.On("Error", "error building additional modules by multiFactories: %w", factoryErr)

	_, err := kernel.BuildFactory(context.Background(), config, logger, []kernel.Option{
		kernel.WithModuleMultiFactory(func(context.Context, cfg.Config, log.Logger) (map[string]kernel.ModuleFactory, error) {
			return nil, factoryErr
		}),
	})

	assert.EqualError(t, err, "can not build kernel factory: error in module factory")
}

func TestFactoryModuleMultiFactoryPanic(t *testing.T) {
	config, logger, _ := createMocks()

	_, err := kernel.BuildFactory(context.Background(), config, logger, []kernel.Option{
		kernel.WithModuleMultiFactory(func(context.Context, cfg.Config, log.Logger) (map[string]kernel.ModuleFactory, error) {
			panic("panic in module multi factory")
		}),
	})

	assert.True(t, strings.Contains(err.Error(), "can not build kernel factory: panic in module multi factory"))
}

func TestFactoryModuleFactoryPanic(t *testing.T) {
	config, logger, _ := createMocks()

	_, err := kernel.BuildFactory(context.Background(), config, logger, []kernel.Option{
		kernel.WithModuleFactory("module", func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
			panic("panic in module factory")
		}),
	})

	assert.True(t, strings.Contains(err.Error(), "can not build kernel factory: panic in module factory"))
}
