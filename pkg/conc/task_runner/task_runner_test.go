package task_runner_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/coffin"
	taskRunner "github.com/justtrackio/gosoline/pkg/conc/task_runner"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/stretchr/testify/assert"
)

func TestTaskRunner(t *testing.T) {
	ctx := appctx.WithContainer(t.Context())
	runner, err := taskRunner.Provide(ctx)
	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(ctx)

	cfn := coffin.New()
	cfn.GoWithContext(ctx, runner.Run)

	var callCount atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		task := kernel.NewModuleFunc(func(ctx context.Context) error {
			callCount.Add(1)
			wg.Done()

			return nil
		})

		if i%2 == 0 {
			err = taskRunner.RunTask(ctx, task)
		} else {
			err = runner.RunTask(ctx, task)
		}
		assert.NoError(t, err)
	}

	wg.Wait()
	cancel()

	err = cfn.Wait()
	assert.NoError(t, err)
	assert.Equal(t, int32(100), callCount.Load())
}
