package main

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/application"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
)

type helloWorldModule struct{}

func (h *helloWorldModule) Run(ctx context.Context) error {
	fmt.Println("Hello World")

	return nil
}

var moduleFactory = func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
	return &helloWorldModule{}, nil
}

func main() {
	app := application.New()
	app.Add("hello", moduleFactory)
	app.Run()
}
