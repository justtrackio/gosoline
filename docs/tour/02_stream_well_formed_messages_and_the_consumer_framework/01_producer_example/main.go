package main

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/application"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/kernel"
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/applike/gosoline/pkg/mdlsub"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/stream"
)

func main() {
	app := application.Default()
	app.Add("producer-module", NewProducerModule)
	app.Run()
}

func NewProducerModule(_ context.Context, config cfg.Config, logger mon.Logger) (kernel.Module, error) {
	modelId := mdl.ModelId{
		Name: "exampleEvent",
	}
	modelId.PadFromConfig(config)
	producer, err := stream.NewProducer(config, logger, "exampleProducer")

	if err != nil {
		return nil, fmt.Errorf("failed to create example producer: %w", err)
	}

	return producerModule{
		modelId:  modelId,
		producer: producer,
		logger:   logger,
	}, nil
}

type producerModule struct {
	modelId  mdl.ModelId
	producer stream.Producer
	logger   mon.Logger
}

func (m producerModule) Run(ctx context.Context) error {
	msg, err := stream.MarshalJsonMessage(map[string]interface{}{
		"greeting": "hello, world",
	}, mdlsub.CreateMessageAttributes(m.modelId, mdlsub.TypeCreate, 0))

	if err != nil {
		return err
	}

	return m.producer.WriteOne(ctx, msg)
}
