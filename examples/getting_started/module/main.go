package main

import (
	"context"
	"time"

	"github.com/justtrackio/gosoline/pkg/application"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
)

func main() {
	application.Run(
		application.WithConfigFile("config.dist.yml", "yml"),
		application.WithModuleFactory("hello-world", NewHelloWorldModule, kernel.ModuleType(kernel.TypeBackground)),
		application.WithModuleFactory("foreground-module", NewForegroundModule, kernel.ModuleType(kernel.TypeForeground)),
	)
}

func NewHelloWorldModule(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
	return &helloWorldModule{
		logger: logger.WithChannel("hello-world"),
	}, nil
}

type helloWorldModule struct {
	logger log.Logger
}

func (h *helloWorldModule) Run(ctx context.Context) error {
	ticker := time.Tick(10 * time.Second)

	select {
	case <-ctx.Done():
		h.logger.Info(ctx, "Time to stop")
	case <-ticker:
		h.logger.Info(ctx, "Hello World")
	}

	return nil
}

func NewForegroundModule(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
	return &foregroundModule{
		logger: logger.WithChannel("foreground-module"),
	}, nil
}

type foregroundModule struct {
	logger log.Logger
}

func (e *foregroundModule) Run(ctx context.Context) error {
	e.logger.Info(ctx, "Foreground module")

	return nil
}
