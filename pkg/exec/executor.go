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
	Executable func(ctx context.Context) (interface{}, error)
	Notify     func(error, time.Duration)
)

type Executor interface {
	Execute(ctx context.Context, f Executable, notifier ...Notify) (interface{}, error)
}

func NewExecutor(logger log.Logger, res *ExecutableResource, settings *BackoffSettings, checks []ErrorChecker, notifier ...Notify) Executor {
	//if !settings.Enabled {
	//	return NewDefaultExecutor()
	//}

	return NewBackoffExecutor(logger, res, settings, checks, notifier...)
}

type DefaultExecutor struct{}

func NewDefaultExecutor() *DefaultExecutor {
	return &DefaultExecutor{}
}

func (e DefaultExecutor) Execute(ctx context.Context, f Executable, notifier ...Notify) (interface{}, error) {
	return f(ctx)
}
