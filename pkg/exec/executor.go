package exec

import (
	"context"
	"github.com/applike/gosoline/pkg/mon"
)

type ExecutableResource struct {
	Type string
	Name string
}

type Executable func(ctx context.Context) (interface{}, error)

type Executor interface {
	Execute(ctx context.Context, f Executable) (interface{}, error)
}

func NewExecutor(logger mon.Logger, res *ExecutableResource, settings *BackoffSettings, checks ...ErrorChecker) Executor {
	if !settings.Enabled {
		return NewDefaultExecutor()
	}

	return NewBackoffExecutor(logger, res, settings, checks...)
}

type DefaultExecutor struct {
}

func NewDefaultExecutor() *DefaultExecutor {
	return &DefaultExecutor{}
}

func (e DefaultExecutor) Execute(ctx context.Context, f Executable) (interface{}, error) {
	return f(ctx)
}
