package main

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/application"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/coffin"
	"github.com/applike/gosoline/pkg/kernel"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/stream"
	"time"
)

func main() {
	app := application.Default()
	app.Add("input-module", NewInputModule)
	app.Run()
}

func NewInputModule(_ context.Context, config cfg.Config, logger mon.Logger) (kernel.Module, error) {
	input, err := stream.NewConfigurableInput(config, logger, "exampleInput")

	if err != nil {
		return nil, fmt.Errorf("failed to create example input: %w", err)
	}

	go provideFakeData()

	return inputModule{
		input:  input,
		logger: logger,
	}, nil
}

type inputModule struct {
	input  stream.Input
	logger mon.Logger
}

func provideFakeData() {
	input := stream.ProvideInMemoryInput("exampleInput", &stream.InMemorySettings{
		Size: 1,
	})

	for msg := "a"; len(msg) <= 10; msg += "a" {
		input.Publish(&stream.Message{
			Body: msg,
		})
	}
}

func (m inputModule) Run(ctx context.Context) error {
	logger := m.logger.WithContext(ctx)
	cfn := coffin.New()

	cfn.GoWithContext(ctx, m.input.Run)
	cfn.Go(func() error {
		consumed := 0

		for item := range m.input.Data() {
			logger.Info("received new message, processing it: %s", item.Body)

			// fake some work...
			time.Sleep(time.Millisecond * 100 * time.Duration(len(item.Body)))

			consumed++

			if consumed == 10 {
				m.input.Stop()
			}
		}

		return nil
	})

	return cfn.Wait()
}
