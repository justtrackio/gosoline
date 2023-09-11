package main

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/application"
)

func NewHelloWorldModule(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
	return &HelloWorldModule{
		// highlight-next-line
		logger: logger.WithChannel("hello-world"),
	}, nil
}

type HelloWorldModule struct {
	// highlight-next-line
	logger log.Logger
}

func (h HelloWorldModule) Run(ctx context.Context) error {
	// highlight-next-line
	h.logger.Info("Hello World")
	return nil
}

func main() {
	application.Run(
		application.WithModuleFactory("hello-world", NewHelloWorldModule),
	)
}