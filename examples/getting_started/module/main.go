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
	for i := 0; i < 3; i++ {
		app.Add(fmt.Sprintf("foreground-module-%d", i), NewForegroundModule, kernel.ModuleType(kernel.TypeForeground))
	}

	app.Add("foregroundErrorModule", NewForegroundErrorModule, kernel.ModuleType(kernel.TypeForeground))

	app.Run()
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
	stop := time.Tick(10 * time.Second)

	for {
		select {
		case <-ctx.Done():
			e.logger.Info("Time to stop, DONE channel")
			return nil
		case <-ticker:
			e.logger.Info("Foreground module - tick")
		case <-stop:
			e.logger.Info("stoping due to 10s timeout")
			return nil
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
	e.logger.Info("Foreground module - NO error")

	return fmt.Errorf("aaa")
}
