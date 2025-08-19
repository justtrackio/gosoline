package main

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/application"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
)

func main() {
	app := application.New(
		application.WithModuleFactory("hello", moduleFactory),
		application.WithMiddlewareFactory(two, kernel.PositionBeginning),
		application.WithMiddlewareFactory(one, kernel.PositionBeginning),
	)
	app.Run()
}

func moduleFactory(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
	return &helloWorldModule{}, nil
}

type helloWorldModule struct{}

func (h *helloWorldModule) Run(ctx context.Context) error {
	fmt.Println("Hello World")

	return nil
}

func one(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Middleware, error) {
	return func(next kernel.MiddlewareHandler) kernel.MiddlewareHandler {
		return func(ctx context.Context) {
			fmt.Println("Beginning of one")

			next(ctx)

			fmt.Println("End of one")
		}
	}, nil
}

func two(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Middleware, error) {
	return func(next kernel.MiddlewareHandler) kernel.MiddlewareHandler {
		return func(ctx context.Context) {
			fmt.Println("Beginning of two")

			next(ctx)

			fmt.Println("End of two")
		}
	}, nil
}
