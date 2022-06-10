package kernel_test

import (
	"context"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestMiddleware(t *testing.T) {
	config, logger, module := createMocks()
	callstack := make([]string, 0)

	module.On("GetStage").Return(kernel.StageApplication)
	module.On("Run", mock.Anything).Run(func(args mock.Arguments) {
		callstack = append(callstack, "module")
	}).Return(nil)

	k, err := kernel.BuildKernel(context.Background(), config, logger, []kernel.Option{
		kernel.WithMiddlewareFactory(kernel.BuildSimpeMiddleware(func(next kernel.MiddlewareHandler) {
			callstack = append(callstack, "mid1 start")
			next()
			callstack = append(callstack, "mid1 end")
		}), kernel.PositionEnd),

		kernel.WithMiddlewareFactory(kernel.BuildSimpeMiddleware(func(next kernel.MiddlewareHandler) {
			callstack = append(callstack, "mid2 start")
			next()
			callstack = append(callstack, "mid2 end")
		}), kernel.PositionEnd),

		kernel.WithMiddlewareFactory(kernel.BuildSimpeMiddleware(func(next kernel.MiddlewareHandler) {
			callstack = append(callstack, "mid3 start")
			next()
			callstack = append(callstack, "mid3 end")
		}), kernel.PositionBeginning),

		kernel.WithModuleFactory("module", func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
			return module, nil
		}),

		kernel.WithKillTimeout(time.Second),
		mockExitHandler(t, kernel.ExitCodeOk),
	})

	assert.NoError(t, err)
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

	assert.Equal(t, expectedCallstack, callstack)
}
