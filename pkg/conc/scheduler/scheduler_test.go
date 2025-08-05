package scheduler_test

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/coffin"
	"github.com/justtrackio/gosoline/pkg/conc/scheduler"
	taskRunner "github.com/justtrackio/gosoline/pkg/conc/task_runner"
	metricMocks "github.com/justtrackio/gosoline/pkg/metric/mocks"
	"github.com/stretchr/testify/assert"
)

func TestScheduler(t *testing.T) {
	ctx := appctx.WithContainer(t.Context())
	runner, err := taskRunner.Provide(ctx)
	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(ctx)

	cfn := coffin.New(ctx)
	cfn.GoWithContext("runner", runner.Run)

	batchRunner := func(ctx context.Context, keys []string, providers []func() (int, error)) (map[string]int, error) {
		results := map[string]int{}
		var err error

		for i, key := range keys {
			results[key], err = providers[i]()
			if err != nil {
				return nil, err
			}
		}

		return results, nil
	}
	metricWriter := metricMocks.NewWriterMockedAll()
	taskScheduler := scheduler.NewSchedulerWithSettings[int](batchRunner, metricWriter, "test", scheduler.Settings{
		BatchTimeout: time.Millisecond * 3,
		RunnerCount:  5,
		MaxBatchSize: 8,
	})
	err = runner.RunTask(taskScheduler)
	assert.NoError(t, err)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		cfn.Go(fmt.Sprintf("tasks %d", i), func() error {
			defer wg.Done()

			for j := i * 10; j < (i+1)*10; j++ {
				jobRunner := func() (int, error) {
					return -j, nil
				}
				result, err := taskScheduler.ScheduleJob(strconv.Itoa(j), jobRunner)
				assert.Equal(t, result, -j)
				assert.NoError(t, err)
			}

			return nil
		})
	}

	wg.Wait()
	cancel()

	err = cfn.Wait()
	assert.NoError(t, err)
}
