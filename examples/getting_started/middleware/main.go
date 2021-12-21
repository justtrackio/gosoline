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
	app := application.New()
	app.Add("hello", moduleFactory)
	app.AddMiddleware(two, kernel.PositionBeginning)
	app.AddMiddleware(one, kernel.PositionBeginning)
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

func one(ctx context.Context, config cfg.Config, logger log.Logger, next kernel.Handler) kernel.Handler {
	return func() {
		fmt.Println("Beginning of one")

		next()

		fmt.Println("End of one")
	}
}

func two(ctx context.Context, config cfg.Config, logger log.Logger, next kernel.Handler) kernel.Handler {
	return func() {
		fmt.Println("Beginning of two")

		next()

		fmt.Println("End of two")
	}
}
