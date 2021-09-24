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

	k, err := kernel.New(context.Background(), config, logger, kernel.KillTimeout(time.Second))
	assert.NoError(t, err)

	callstack := make([]string, 0)

	module.On("GetStage").Return(kernel.StageApplication)
	module.On("Run", mock.Anything).Run(func(args mock.Arguments) {
		callstack = append(callstack, "module")
	}).Return(nil)

	k.Add("module", func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
		return module, nil
	})

	k.AddMiddleware(func(ctx context.Context, config cfg.Config, logger log.Logger, next kernel.Handler) kernel.Handler {
		return func() {
			callstack = append(callstack, "mid1 start")
			next()
			callstack = append(callstack, "mid1 end")
		}
	}, kernel.PositionEnd)

	k.AddMiddleware(func(ctx context.Context, config cfg.Config, logger log.Logger, next kernel.Handler) kernel.Handler {
		return func() {
			callstack = append(callstack, "mid2 start")
			next()
			callstack = append(callstack, "mid2 end")
		}
	}, kernel.PositionEnd)

	k.AddMiddleware(func(ctx context.Context, config cfg.Config, logger log.Logger, next kernel.Handler) kernel.Handler {
		return func() {
			callstack = append(callstack, "mid3 start")
			next()
			callstack = append(callstack, "mid3 end")
		}
	}, kernel.PositionBeginning)

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
