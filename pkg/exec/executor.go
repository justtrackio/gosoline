package exec

import (
	"context"
)

type ExecutableResource struct {
	Type string
	Name string
}

type Executable func(ctx context.Context) (interface{}, error)

type Executor interface {
	Execute(ctx context.Context, f Executable) (interface{}, error)
}

type DefaultExecutor struct {
}

func NewDefaultExecutor() *DefaultExecutor {
	return &DefaultExecutor{}
}

func (e DefaultExecutor) Execute(ctx context.Context, f Executable) (interface{}, error) {
	return f(ctx)
}
