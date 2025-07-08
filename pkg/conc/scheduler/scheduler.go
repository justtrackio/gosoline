package scheduler

import (
	"context"
	"fmt"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/coffin"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/metric"
)

const (
	metricNameBatchSize = "schedulerBatchSize"
	metricNameTaskDelay = "schedulerTaskDelay"
)

//go:generate go run github.com/vektra/mockery/v2 --name Scheduler
type Scheduler[T any] interface {
	ScheduleJob(key string, provider func() (T, error)) (T, error)
	Run(ctx context.Context) error
}

type BatchRunner[T any] func(ctx context.Context, keys []string, providers []func() (T, error)) (map[string]T, error)

type result[T any] struct {
	success *T
	failure error
}

type job[T any] struct {
	key      string
	provider func() (T, error)
	result   chan result[T]
}

type jobBatch[T any] struct {
	keys      []string
	providers []func() (T, error)
	results   []chan result[T]
}

type scheduler[T any] struct {
	batchRunner  BatchRunner[T]
	batchTimeout time.Duration
	clock        clock.Clock
	metricWriter metric.Writer
	runnerCount  int
	maxBatchSize int
	name         string
	workQueue    chan job[T]
}

type Settings struct {
	BatchTimeout time.Duration `cfg:"batch_timeout" default:"10ms"`
	RunnerCount  int           `cfg:"runner_count" default:"25"`
	MaxBatchSize int           `cfg:"max_batch_size" default:"25"`
}

func NewScheduler[T any](config cfg.Config, batchRunner BatchRunner[T], name string) Scheduler[T] {
	var settings Settings
	config.UnmarshalKey(fmt.Sprintf("scheduler.%s", name), &settings)

	metricWriter := metric.NewWriter(getDefaultMetrics(name)...)

	return NewSchedulerWithSettings[T](batchRunner, metricWriter, name, settings)
}

func NewSchedulerWithSettings[T any](batchRunner BatchRunner[T], metricWriter metric.Writer, name string, settings Settings) Scheduler[T] {
	return &scheduler[T]{
		batchRunner:  batchRunner,
		batchTimeout: settings.BatchTimeout,
		clock:        clock.Provider,
		metricWriter: metricWriter,
		runnerCount:  settings.RunnerCount,
		maxBatchSize: settings.MaxBatchSize,
		name:         name,
		workQueue:    make(chan job[T], settings.RunnerCount),
	}
}

func (s scheduler[T]) ScheduleJob(key string, provider func() (T, error)) (T, error) {
	start := s.clock.Now()
	resultChan := make(chan result[T])
	s.workQueue <- job[T]{
		key:      key,
		provider: provider,
		result:   resultChan,
	}

	resultValue := <-resultChan
	s.writeTaskDelayMetric(s.clock.Since(start))

	if resultValue.success != nil {
		return *resultValue.success, nil
	}

	return mdl.Empty[T](), resultValue.failure
}

func (s scheduler[T]) Run(ctx context.Context) error {
	cfn := coffin.New()
	cfn.Go(func() error {
		batchQueue := make(chan jobBatch[T], s.runnerCount)

		cfn.Go(func() error {
			return s.createBatches(ctx, batchQueue)
		})

		// execute batches of jobs in parallel
		for i := 0; i < s.runnerCount; i++ {
			cfn.Go(func() error {
				return s.executeBatches(ctx, batchQueue)
			})
		}

		return nil
	})

	return cfn.Wait()
}

func (s scheduler[T]) createBatches(ctx context.Context, batchQueue chan jobBatch[T]) error {
	defer close(batchQueue)

	currentBatch := jobBatch[T]{}
	ticker := s.clock.NewTicker(time.Hour)
	// immediately stop the ticker as we only need it running when we have a batch open
	ticker.Stop()
	// ensure the ticker is stopped once we are done
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// flush batch if needed
			if len(currentBatch.keys) > 0 {
				batchQueue <- currentBatch
			}

			return nil
		case <-ticker.Chan():
			// flush batch if needed
			if len(currentBatch.keys) > 0 {
				batchQueue <- currentBatch
				currentBatch = jobBatch[T]{}
			}
			// stop it again as we only need it while a batch is being formed
			ticker.Stop()
		case job := <-s.workQueue:
			currentBatch.keys = append(currentBatch.keys, job.key)
			currentBatch.providers = append(currentBatch.providers, job.provider)
			currentBatch.results = append(currentBatch.results, job.result)

			// start the timer after receiving the first record
			if len(currentBatch.keys) == 1 {
				ticker.Reset(s.batchTimeout)
			}

			if len(currentBatch.keys) == s.maxBatchSize {
				// flush batch
				batchQueue <- currentBatch
				currentBatch = jobBatch[T]{}
				// and stop the ticker until we receive a tick again
				ticker.Stop()
			}
		}
	}
}

func (s scheduler[T]) executeBatches(ctx context.Context, batchQueue chan jobBatch[T]) error {
	for batch := range batchQueue {
		s.writeBatchSizeMetric(len(batch.keys))

		results, err := s.batchRunner(ctx, batch.keys, batch.providers)
		if err != nil {
			for _, resultChan := range batch.results {
				resultChan <- result[T]{
					failure: err,
				}
			}
		} else {
			for i, resultChan := range batch.results {
				resultChan <- result[T]{
					success: mdl.Box(results[batch.keys[i]]),
				}
			}
		}
	}

	return nil
}

func (s scheduler[T]) writeTaskDelayMetric(took time.Duration) {
	s.metricWriter.WriteOne(&metric.Datum{
		Priority:   metric.PriorityHigh,
		MetricName: metricNameTaskDelay,
		Dimensions: map[string]string{
			"Scheduler": s.name,
		},
		Value: float64(took.Milliseconds()),
		Unit:  metric.UnitMillisecondsAverage,
	})
}

func (s scheduler[T]) writeBatchSizeMetric(batchSize int) {
	s.metricWriter.WriteOne(&metric.Datum{
		Priority:   metric.PriorityHigh,
		MetricName: metricNameBatchSize,
		Dimensions: map[string]string{
			"Scheduler": s.name,
		},
		Value: float64(batchSize),
		Unit:  metric.UnitCountAverage,
	})
}

func getDefaultMetrics(name string) metric.Data {
	return metric.Data{
		{
			Priority:   metric.PriorityHigh,
			MetricName: metricNameTaskDelay,
			Dimensions: map[string]string{
				"Scheduler": name,
			},
			Value: 0,
			Unit:  metric.UnitMillisecondsAverage,
		},
		{
			Priority:   metric.PriorityHigh,
			MetricName: metricNameBatchSize,
			Dimensions: map[string]string{
				"Scheduler": name,
			},
			Value: 0,
			Unit:  metric.UnitCountAverage,
		},
	}
}
