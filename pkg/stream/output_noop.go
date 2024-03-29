package stream

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

type NoOpOutput struct{}

func newNoOpOutput(_ context.Context, _ cfg.Config, _ log.Logger, _ string) (Output, error) {
	return &NoOpOutput{}, nil
}

func (o *NoOpOutput) WriteOne(_ context.Context, _ WritableMessage) error {
	return nil
}

func (o *NoOpOutput) Write(_ context.Context, _ []WritableMessage) error {
	return nil
}
