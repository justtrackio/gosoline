package main

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/application"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/stream"
	"time"
)

func main() {
	application.RunConsumer(NewConsumerCallback)
}

func NewConsumerCallback(_ context.Context, _ cfg.Config, logger mon.Logger) (stream.ConsumerCallback, error) {
	go provideFakeData()

	return consumerCallback{
		logger: logger,
	}, nil
}

type consumerCallback struct {
	logger mon.Logger
}

func provideFakeData() {
	input := stream.ProvideInMemoryInput("exampleInput", &stream.InMemorySettings{
		Size: 1,
	})

	input.Publish(&stream.Message{
		Attributes: map[string]interface{}{
			"modelId": "gosoline.stream-example.example.printCommand",
		},
		Body: `{"message":"hello, world"}`,
	})
	input.Publish(&stream.Message{
		Attributes: map[string]interface{}{
			"modelId": "gosoline.stream-example.example.waitCommand",
		},
		Body: `{"time":1}`,
	})
	input.Publish(&stream.Message{
		Attributes: map[string]interface{}{
			"modelId": "gosoline.stream-example.example.printCommand",
		},
		Body: `{"message":"processing..."}`,
	})
	input.Publish(&stream.Message{
		Attributes: map[string]interface{}{
			"modelId": "gosoline.stream-example.example.waitCommand",
		},
		Body: `{"time":3}`,
	})
	input.Publish(&stream.Message{
		Attributes: map[string]interface{}{
			"modelId": "gosoline.stream-example.example.printCommand",
		},
		Body: `{"message":"bye"}`,
	})

	input.Stop()
}

type PrintCommand struct {
	Message string `json:"message"`
}

type WaitCommand struct {
	Time int `json:"time"`
}

func (c consumerCallback) GetModel(attributes map[string]interface{}) interface{} {
	switch attributes["modelId"] {
	case "gosoline.stream-example.example.printCommand":
		return &PrintCommand{}
	case "gosoline.stream-example.example.waitCommand":
		return &WaitCommand{}
	default:
		return nil
	}
}

func (c consumerCallback) Consume(ctx context.Context, model interface{}, attributes map[string]interface{}) (bool, error) {
	switch cmd := model.(type) {
	case *PrintCommand:
		c.logger.WithContext(ctx).Info("printing message: %s", cmd.Message)

		return true, nil
	case *WaitCommand:
		time.Sleep(time.Duration(cmd.Time) * time.Second)

		return true, nil
	default:
		return true, fmt.Errorf("unknown model: %s with type %T", attributes["modelId"], model)
	}
}
