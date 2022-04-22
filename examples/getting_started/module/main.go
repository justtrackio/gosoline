package main

import (
	"context"
	"fmt"
	"time"

	"github.com/justtrackio/gosoline/pkg/application"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
)

func main() {
	app := application.Default()
	app.Add("hello-world", NewHelloWorldModule, kernel.ModuleType(kernel.TypeBackground))
	for i := 0; i < 2; i++ {
		app.Add(fmt.Sprintf("foreground-module-%d", i), NewForegroundModule, kernel.ModuleType(kernel.TypeForeground))
	}

	app.Add("foregroundErrorModule", NewForegroundErrorModule, kernel.ModuleType(kernel.TypeBackground))

	app.Run()
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
	ticker := time.Tick(2 * time.Second)

	select {
	case <-ctx.Done():
		h.logger.Info("Time to stop @@@@@@@@@@@@@@@@@@@@")
	case <-ticker:
		h.logger.Info("Hello World")
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
	e.logger.Info("Foreground module")
	ticker := time.Tick(1 * time.Second)

	for {
		select {
		case <-ctx.Done():
			e.logger.Info("######## ######## Time to stop")
			// return nil
		case <-ticker:
			e.logger.Info("Foreground module - tick")
		}
	}

	return nil
}

func NewForegroundErrorModule(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
	return &foregroundErrorModule{
		logger: logger.WithChannel("foreground-error-module"),
	}, nil
}

type foregroundErrorModule struct {
	logger log.Logger
}

func (e *foregroundErrorModule) Run(ctx context.Context) error {
	e.logger.Info("Foreground module - mocked error")

	time.Sleep(4 * time.Second)

	return fmt.Errorf("mocked error")
}
