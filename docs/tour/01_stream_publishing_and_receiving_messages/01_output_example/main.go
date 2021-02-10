package main

import (
	"context"
	"github.com/applike/gosoline/pkg/application"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/kernel"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/stream"
)

func main() {
	app := application.Default()
	app.Add("output-module", NewOutputModule)
	app.Run()
}

func NewOutputModule(_ context.Context, config cfg.Config, logger mon.Logger) (kernel.Module, error) {
	output := stream.NewConfigurableOutput(config, logger, "exampleOutput")

	return outputModule{
		output: output,
	}, nil
}

type outputModule struct {
	output stream.Output
}

func (m outputModule) Run(ctx context.Context) error {
	msg := stream.NewRawJsonMessage(map[string]interface{}{
		"greeting": "hello, world",
	})

	return m.output.WriteOne(ctx, msg)
}
