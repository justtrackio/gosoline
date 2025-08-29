package main

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
)

func NewHelloWorldModule(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
	return &HelloWorldModule{
		logger: logger.WithChannel("hello-world"),
	}, nil
}

type HelloWorldModule struct {
	logger log.Logger
}

func (h HelloWorldModule) Run(ctx context.Context) error {
	h.logger.Info(ctx, "Hello World")

	return nil
}
