package main

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/application"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/kernel"
	"github.com/applike/gosoline/pkg/mdlsub"
	"github.com/applike/gosoline/pkg/mon"
)

func main() {
	app := application.Default()
	app.Add("publisher-module", NewProducerModule)
	app.Run()
}

func NewProducerModule(_ context.Context, config cfg.Config, logger mon.Logger) (kernel.Module, error) {
	publisher, err := mdlsub.NewPublisher(config, logger, "examplePublisher")

	if err != nil {
		return nil, fmt.Errorf("failed to create example publisher: %w", err)
	}

	return publisherModule{
		publisher: publisher,
		logger:    logger,
	}, nil
}

type publisherModule struct {
	kernel.EssentialModule
	publisher mdlsub.Publisher
	logger    mon.Logger
}

type ExampleMessage struct {
	Greeting string `json:"greeting"`
}

func (m publisherModule) Run(ctx context.Context) error {
	return m.publisher.Publish(ctx, mdlsub.TypeCreate, 0, &ExampleMessage{
		Greeting: "hello, world",
	})
}
