package aws

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/exec"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/aws/aws-sdk-go/aws/request"
	"net/http"
)

type RequestFunction func() (*request.Request, interface{})

//go:generate mockery -name Executor
type Executor interface {
	Execute(ctx context.Context, f RequestFunction) (interface{}, error)
}

func NewExecutor(logger mon.Logger, res *exec.ExecutableResource, settings *exec.BackoffSettings, checks ...exec.ErrorChecker) Executor {
	if !settings.Enabled {
		return new(DefaultExecutor)
	}

	return NewBackoffExecutor(logger, res, settings, checks...)
}

type DefaultExecutor struct {
}

func (e DefaultExecutor) Execute(ctx context.Context, f RequestFunction) (interface{}, error) {
	req, out := f()

	req.SetContext(ctx)
	err := req.Send()

	return out, err
}

type TestExecution struct {
	Output interface{}
	Err    error
}

type TestableExecutor struct {
	executions []TestExecution
	current    int
}

func NewTestableExecutor(executions []TestExecution) *TestableExecutor {
	return &TestableExecutor{
		executions: executions,
	}
}

func (t *TestableExecutor) Execute(ctx context.Context, f RequestFunction) (interface{}, error) {
	f()

	c := t.current
	t.current++

	return t.executions[c].Output, t.executions[c].Err
}

type BackoffExecutor struct {
	executor exec.Executor
}

func NewBackoffExecutor(logger mon.Logger, res *exec.ExecutableResource, settings *exec.BackoffSettings, checks ...exec.ErrorChecker) Executor {
	checks = append(checks, []exec.ErrorChecker{
		exec.CheckRequestCanceled,
		exec.CheckUsedClosedConnectionError,
		CheckConnectionError,
		CheckErrorRetryable,
		CheckErrorThrottle,
	}...)

	return &BackoffExecutor{
		executor: exec.NewBackoffExecutor(logger, res, settings, checks...),
	}
}

func (b BackoffExecutor) Execute(ctx context.Context, f RequestFunction) (interface{}, error) {
	return b.executor.Execute(ctx, func(ctx context.Context) (interface{}, error) {
		req, out := f()

		req.SetContext(ctx)
		err := req.Send()

		if req.HTTPResponse.StatusCode >= http.StatusInternalServerError && req.HTTPResponse.StatusCode != http.StatusNotImplemented {
			return nil, fmt.Errorf("http status code: %d", req.HTTPResponse.StatusCode)
		}

		return out, err
	})
}
