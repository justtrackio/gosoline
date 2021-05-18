package stream

import (
	"context"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
)

type NoOpOutput struct {
}

func newNoOpOutput(_ cfg.Config, _ mon.Logger, _ string) (Output, error) {
	return &NoOpOutput{}, nil
}

func (o *NoOpOutput) WriteOne(_ context.Context, _ WritableMessage) error {
	return nil
}

func (o *NoOpOutput) Write(_ context.Context, _ []WritableMessage) error {
	return nil
}
