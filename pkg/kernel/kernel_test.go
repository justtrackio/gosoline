package kernel_test

import (
	"context"
	"errors"
	cfgMocks "github.com/applike/gosoline/pkg/cfg/mocks"
	"github.com/applike/gosoline/pkg/kernel"
	kernelMocks "github.com/applike/gosoline/pkg/kernel/mocks"
	monMocks "github.com/applike/gosoline/pkg/mon/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

func createMocks() (*cfgMocks.Config, *monMocks.Logger, *kernelMocks.Module) {
	config := new(cfgMocks.Config)
	config.On("AllKeys").Return([]string{})

	logger := new(monMocks.Logger)
	logger.On("WithChannel", mock.Anything).Return(logger)
	logger.On("WithFields", mock.Anything).Return(logger)
	logger.On("Info", mock.Anything)
	logger.On("Infof", mock.Anything, mock.Anything)

	module := new(kernelMocks.Module)
	module.On("GetType").Return(kernel.TypeForeground)

	return config, logger, module
}

func TestRunSuccess(t *testing.T) {
	config, logger, module := createMocks()

	module.On("Boot", config, logger).Return(nil)
	module.On("Run", mock.Anything).Return(nil)

	assert.NotPanics(t, func() {
		k := kernel.NewWithInterfaces(config, logger)
		k.Add("module", module)
		k.Run()
	})

	module.AssertCalled(t, "Boot", mock.Anything, mock.Anything)
	module.AssertCalled(t, "Run", mock.Anything)
}

func TestBootFailure(t *testing.T) {
	config, logger, module := createMocks()

	failure1 := errors.New("error booting module module")
	failure2 := errors.New("could not boot module due to a panic")

	logger.On("Info", mock.Anything)
	logger.On("Error", failure1, "error booting module module")
	logger.On("Error", failure2, "error during the boot process of the kernel")

	module.On("Boot", config, logger).Run(func(args mock.Arguments) {
		panic(failure1)
	}).Return(nil)

	assert.NotPanics(t, func() {
		k := kernel.NewWithInterfaces(config, logger)
		k.Add("module", module)
		k.Run()
	})

	module.AssertCalled(t, "Boot", mock.Anything, mock.Anything)
}

func TestRunFailure(t *testing.T) {
	failure := errors.New("panic in run")
	config, logger, module := createMocks()
	logger.On("Error", failure, "error running module module")
	logger.On("Error", errors.New("error running module module"), "error during the execution of the kernel")

	module.On("Boot", config, logger).Return(nil)
	module.On("Run", mock.Anything).Run(func(args mock.Arguments) {
		panic(failure)
	})

	assert.NotPanics(t, func() {
		k := kernel.NewWithInterfaces(config, logger)
		k.Add("module", module)
		k.Run()
	})

	module.AssertCalled(t, "Boot", mock.Anything, mock.Anything)
	module.AssertCalled(t, "Run", mock.Anything)
}

func TestStop(t *testing.T) {
	config, logger, module := createMocks()
	k := kernel.NewWithInterfaces(config, logger)

	module.On("GetType").Return(kernel.TypeForeground)
	module.On("Boot", config, logger).Return(nil)
	module.On("Run", mock.Anything).Run(func(args mock.Arguments) {
		ctx := args.Get(0).(context.Context)

		k.Stop("test done")
		<-ctx.Done()
	}).Return(nil)

	k.Add("module", module)
	k.Run()

	module.AssertCalled(t, "Boot", mock.Anything, mock.Anything)
	module.AssertCalled(t, "Run", mock.Anything)
}

func TestRunningType(t *testing.T) {
	config, logger, _ := createMocks()
	k := kernel.NewWithInterfaces(config, logger)

	mf := new(kernelMocks.Module)
	mf.On("GetType").Return(kernel.TypeForeground)
	mf.On("Boot", config, logger).Return(nil)
	mf.On("Run", mock.Anything).Run(func(args mock.Arguments) {}).Return(nil)

	mb := new(kernelMocks.Module)
	mb.On("GetType").Return(kernel.TypeBackground)
	mb.On("Boot", config, logger).Return(nil)
	mb.On("Run", mock.Anything).Run(func(args mock.Arguments) {
		ctx := args.Get(0).(context.Context)

		<-ctx.Done()
	}).Return(nil)

	k.Add("foreground", mf)
	k.Add("background", mb)
	k.Run()

	mf.AssertExpectations(t)
	mb.AssertExpectations(t)
}
