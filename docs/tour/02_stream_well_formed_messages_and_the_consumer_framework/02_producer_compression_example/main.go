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

func oneMillionLOLs() string {
	lol := "LOL"

	for i := 0; i < 20; i++ {
		lol = lol + lol
	}

	return lol
}

func (m producerModule) Run(ctx context.Context) error {
	greeting := oneMillionLOLs()
	msg, err := stream.MarshalJsonMessage(map[string]interface{}{
		"greeting": greeting,
	}, mdlsub.CreateMessageAttributes(m.modelId, mdlsub.TypeCreate, 0))

	if err != nil {
		return err
	}

	m.logger.WithContext(ctx).Info("publishing a message with more than %d characters", len(greeting))

	defer func() {
		output := stream.ProvideInMemoryOutput("exampleProducer")
		msg, ok := output.Get(0)

		if ok {
			m.logger.WithContext(ctx).Info("published message with encoded body length of %d characters, attributes %v", len(msg.Body), msg.Attributes)
		}
	}()

	return m.producer.WriteOne(ctx, msg)
}
