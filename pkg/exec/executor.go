package exec

import (
	"context"
	"fmt"
	"time"

	"github.com/justtrackio/gosoline/pkg/log"
)

type ExecutableResource struct {
	Type string
	Name string
}

func (r ExecutableResource) String() string {
	return fmt.Sprintf("%s/%s", r.Type, r.Name)
}

type (
	Executable func(ctx context.Context) (any, error)
	Notify     func(error, time.Duration)
)

type Executor interface {
	Execute(ctx context.Context, f Executable, notifier ...Notify) (any, error)
}

func NewExecutor(logger log.Logger, res *ExecutableResource, settings *BackoffSettings, checks []ErrorChecker, notifier ...Notify) Executor {
	return NewBackoffExecutor(logger, res, settings, checks, notifier...)
}

type DefaultExecutor struct{}

func NewDefaultExecutor() Executor {
	return &DefaultExecutor{}
}

func (e DefaultExecutor) Execute(ctx context.Context, f Executable, notifier ...Notify) (any, error) {
	return f(ctx)
}
