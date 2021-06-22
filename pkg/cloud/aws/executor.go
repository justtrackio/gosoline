package aws

import (
	"context"
	"github.com/applike/gosoline/pkg/exec"
	"github.com/applike/gosoline/pkg/log"
	"github.com/aws/aws-sdk-go/aws/request"
	"net/http"
)

type RequestFunction func() (*request.Request, interface{})

//go:generate mockery -name Executor
type Executor interface {
	Execute(ctx context.Context, f RequestFunction) (interface{}, error)
}

func NewExecutor(logger log.Logger, res *exec.ExecutableResource, settings *exec.BackoffSettings, checks ...exec.ErrorChecker) Executor {
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

type Sender func(req *request.Request) (*http.Response, error)

type BackoffExecutor struct {
	executor exec.Executor
	sender   Sender
}

func NewBackoffExecutor(logger log.Logger, res *exec.ExecutableResource, settings *exec.BackoffSettings, checks ...exec.ErrorChecker) Executor {
	return NewBackoffExecutorWithSender(logger, res, settings, func(req *request.Request) (*http.Response, error) {
		err := req.Send()

		return req.HTTPResponse, err
	}, checks...)
}

func NewBackoffExecutorWithSender(logger log.Logger, res *exec.ExecutableResource, settings *exec.BackoffSettings, sender Sender, checks ...exec.ErrorChecker) Executor {
	checks = append(checks, []exec.ErrorChecker{
		exec.CheckRequestCanceled,
		exec.CheckUsedClosedConnectionError,
		exec.CheckTimeoutError,
		exec.CheckClientAwaitHeaderTimeoutError,
		exec.CheckTlsHandshakeTimeoutError,
		CheckInvalidStatusError,
		CheckConnectionError,
		CheckErrorRetryable,
		CheckErrorThrottle,
	}...)

	return &BackoffExecutor{
		executor: exec.NewBackoffExecutor(logger, res, settings, checks...),
		sender:   sender,
	}
}

func (b BackoffExecutor) Execute(ctx context.Context, f RequestFunction) (interface{}, error) {
	return b.executor.Execute(ctx, func(ctx context.Context) (interface{}, error) {
		req, out := f()

		req.SetContext(ctx)
		res, err := b.sender(req)

		// ignore the error should we get a http internal server back, otherwise we do not retry correctly
		if res != nil && res.StatusCode >= http.StatusInternalServerError && res.StatusCode != http.StatusNotImplemented {
			return nil, &InvalidStatusError{
				Status: res.StatusCode,
			}
		}

		return out, err
	})
}
