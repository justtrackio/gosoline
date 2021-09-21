package exec

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/log"
)

type ExecutableResource struct {
	Type string
	Name string
}

func (r ExecutableResource) String() string {
	return fmt.Sprintf("%s/%s", r.Type, r.Name)
}

type Executable func(ctx context.Context) (interface{}, error)

type Executor interface {
	Execute(ctx context.Context, f Executable) (interface{}, error)
}

func NewExecutor(logger log.Logger, res *ExecutableResource, settings *BackoffSettings, checks ...ErrorChecker) Executor {
	//if !settings.Enabled {
	//	return NewDefaultExecutor()
	//}

	return NewBackoffExecutor(logger, res, settings, checks...)
}

type DefaultExecutor struct{}

func NewDefaultExecutor() *DefaultExecutor {
	return &DefaultExecutor{}
}

func (e DefaultExecutor) Execute(ctx context.Context, f Executable) (interface{}, error) {
	return f(ctx)
}
