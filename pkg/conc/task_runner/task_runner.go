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
)

// A TaskRunner allows you to execute tasks instead of adding modules to the kernel. In general, you should give your
// modules a proper name and place in the kernel. This module provides an escape hatch if you need to add something to
// the kernel from outside gosoline (and don't want to add additional code to every service using your code).
//
// Try to avoid using this module if possible. It is only meant as a last resort.
//
//go:generate mockery --name TaskRunner
type TaskRunner interface {
	kernel.Module
	RunTask(task kernel.Module) error
}

type Settings struct {
	Enabled bool `cfg:"enabled" default:"true"`
}

type taskRunner struct {
	lck          sync.Mutex
	done         bool
	pendingTasks chan kernel.Module
}

type taskRunnerKey int

func Factory(_ context.Context, config cfg.Config, _ log.Logger) (map[string]kernel.ModuleFactory, error) {
	var settings Settings
	config.UnmarshalKey("task_runner", &settings)

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
	return appctx.Provide(ctx, taskRunnerKey(0), newTaskRunner)
}

// RunTask gets the TaskRunner from the context and uses it to run the given task.
func RunTask(ctx context.Context, task kernel.Module) error {
	taskRunner, err := Provide(ctx)
	if err != nil {
		return fmt.Errorf("could not find task runner: %w", err)
	}

	err = taskRunner.RunTask(task)
	if err != nil {
		return fmt.Errorf("could not run task on task runner: %w", err)
	}

	return nil
}

func newTaskRunner() (TaskRunner, error) {
	return &taskRunner{
		pendingTasks: make(chan kernel.Module, 100),
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
					cfn.GoWithContext(ctx, task.Run)
				}

				return nil
			case task := <-s.pendingTasks:
				cfn.GoWithContext(ctx, task.Run)
			}
		}
	})

	return cfn.Wait()
}

func (s *taskRunner) RunTask(task kernel.Module) error {
	s.lck.Lock()
	defer s.lck.Unlock()
	if s.done {
		return fmt.Errorf("failed to run task, task runner is already done")
	}

	s.pendingTasks <- task

	return nil
}
