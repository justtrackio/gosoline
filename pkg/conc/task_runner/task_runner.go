package task_runner

import (
	"context"
	"fmt"
	"sync"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/coffin"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/metric"
)

const (
	MetadataKeyTaskRunner = "task_runner"
	metricScheduledTasks  = "task_runner_scheduled_tasks"
	metricStartedTasks    = "task_runner_started_tasks"
	metricFinishedTasks   = "task_runner_finished_tasks"
	metricFailedTasks     = "task_runner_failed_tasks"
)

// A TaskRunner allows you to execute tasks instead of adding modules to the kernel. In general, you should give your
// modules a proper name and place in the kernel. This module provides an escape hatch if you need to add something to
// the kernel from outside gosoline (and don't want to add additional code to every service using your code).
//
// Try to avoid using this module if possible. It is only meant as a last resort.
//
// Enable it by setting task_runner.enabled = true
//
//go:generate go run github.com/vektra/mockery/v2 --name TaskRunner
type TaskRunner interface {
	kernel.Module
	// RunTask schedules a new task to be executed in the background. The context passed to RunTask will be used to
	// schedule the task and is not the context passed to the task itself during execution (as it might already have
	// ended at that point, e.g., when scheduling tasks from an HTTP handler).
	RunTask(ctx context.Context, task kernel.Module) error
}

type Settings struct {
	Enabled bool `cfg:"enabled" default:"false"`
}

type taskRunner struct {
	kernel.EssentialBackgroundModule
	kernel.EssentialStage
	lck          sync.Mutex
	done         bool
	pendingTasks chan kernel.Module
	metricWriter metric.Writer
}

type taskRunnerKey int

func Factory(ctx context.Context, config cfg.Config, _ log.Logger) (map[string]kernel.ModuleFactory, error) {
	var settings Settings
	if err := config.UnmarshalKey("task_runner", &settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal task runner settings: %w", err)
	}

	metadata := map[string]bool{
		"enabled": settings.Enabled,
	}

	if err := appctx.MetadataAppend(ctx, MetadataKeyTaskRunner, metadata); err != nil {
		return nil, fmt.Errorf("can not access the appctx metadata: %w", err)
	}

	if !settings.Enabled {
		return nil, nil
	}

	return map[string]kernel.ModuleFactory{
		"taskRunner": New,
	}, nil
}

func New(ctx context.Context, _ cfg.Config, _ log.Logger) (kernel.Module, error) {
	return Provide(ctx)
}

func Provide(ctx context.Context) (TaskRunner, error) {
	runner, err := appctx.Provide(ctx, taskRunnerKey(0), newTaskRunner)
	if err != nil {
		return nil, fmt.Errorf("failed to provide task runner: %w", err)
	}

	runner.lck.Lock()
	defer runner.lck.Unlock()

	if runner.done {
		runner.done = false
		runner.pendingTasks = make(chan kernel.Module, 100)
	}

	return runner, nil
}

// RunTask gets the TaskRunner from the context and uses it to run the given task.
func RunTask(ctx context.Context, task kernel.Module) error {
	taskRunner, err := Provide(ctx)
	if err != nil {
		return fmt.Errorf("could not find task runner: %w", err)
	}

	err = taskRunner.RunTask(ctx, task)
	if err != nil {
		return fmt.Errorf("could not run task on task runner: %w", err)
	}

	return nil
}

func newTaskRunner() (*taskRunner, error) {
	metricWriter := metric.NewWriter(getMetricDefaults()...)

	return &taskRunner{
		pendingTasks: make(chan kernel.Module, 100),
		metricWriter: metricWriter,
	}, nil
}

func (s *taskRunner) Run(ctx context.Context) error {
	cfn := coffin.New()
	cfn.GoWithContext(ctx, func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				// close the channel and schedule all remaining pending tasks (we promised to run them after all)
				s.lck.Lock()
				s.done = true
				close(s.pendingTasks)
				s.lck.Unlock()

				for task := range s.pendingTasks {
					cfn.GoWithContext(ctx, s.executeTask(task.Run))
				}

				return nil
			case task := <-s.pendingTasks:
				cfn.GoWithContext(ctx, s.executeTask(task.Run))
			}
		}
	})

	return cfn.Wait()
}

func (s *taskRunner) RunTask(ctx context.Context, task kernel.Module) error {
	s.lck.Lock()
	defer s.lck.Unlock()
	if s.done {
		return fmt.Errorf("failed to run task, task runner is already done")
	}

	s.pendingTasks <- task
	s.countTask(ctx, metricScheduledTasks)

	return nil
}

func (s *taskRunner) executeTask(task func(ctx context.Context) error) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		s.countTask(ctx, metricStartedTasks)

		err := task(ctx)
		if err != nil {
			s.countTask(ctx, metricFailedTasks)
		} else {
			s.countTask(ctx, metricFinishedTasks)
		}

		return err
	}
}

func (s *taskRunner) countTask(ctx context.Context, metricName string) {
	s.metricWriter.WriteOne(ctx, &metric.Datum{
		Priority:   metric.PriorityHigh,
		MetricName: metricName,
		Value:      1,
		Unit:       metric.UnitCount,
	})
}

func getMetricDefaults() metric.Data {
	return metric.Data{
		{
			Priority:   metric.PriorityHigh,
			MetricName: metricScheduledTasks,
			Value:      0,
			Unit:       metric.UnitCount,
		},
		{
			Priority:   metric.PriorityHigh,
			MetricName: metricStartedTasks,
			Value:      0,
			Unit:       metric.UnitCount,
		},
		{
			Priority:   metric.PriorityHigh,
			MetricName: metricFailedTasks,
			Value:      0,
			Unit:       metric.UnitCount,
		},
		{
			Priority:   metric.PriorityHigh,
			MetricName: metricFinishedTasks,
			Value:      0,
			Unit:       metric.UnitCount,
		},
	}
}
