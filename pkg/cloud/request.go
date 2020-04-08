package cloud

import (
	"context"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/aws/aws-sdk-go/aws/request"
)

type RequestFunction func() (*request.Request, interface{})

type RequestExecutor interface {
	Execute(ctx context.Context, f RequestFunction) (interface{}, error)
}

func NewExecutor(logger mon.Logger, res *BackoffResource, settings *BackoffSettings) RequestExecutor {
	if !settings.Enabled {
		return new(DefaultExecutor)
	}

	return NewBackoffExecutor(logger, res, settings)
}

type DefaultExecutor struct {
}

func (e DefaultExecutor) Execute(ctx context.Context, f RequestFunction) (interface{}, error) {
	req, out := f()

	req.SetContext(ctx)
	err := req.Send()

	return out, err
}
