package output

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/encoding/json"
	kafkaProducer "github.com/justtrackio/gosoline/pkg/kafka/producer"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/stream"
)

type TestEvent struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

type outputModule struct {
	output stream.Output
}

func NewOutputModule(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
	settings := &kafkaProducer.Settings{
		Compression: kafkaProducer.CompressionGZip,
		Connection:  "default",
		Topic:       "gosoline-test-test-grp-testEvent",
	}

	output, err := stream.NewKafkaOutput(ctx, config, logger, settings)
	if err != nil {
		return nil, fmt.Errorf("can not create kafka output: %w", err)
	}

	return &outputModule{
		output: output,
	}, nil
}

func (p outputModule) Run(ctx context.Context) error {
	event := &TestEvent{
		Id:   123,
		Name: fmt.Sprintf("event %d", 123),
	}

	eventMarshalled, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("can not marshal event: %w", err)
	}

	msg := &stream.Message{
		Body: string(eventMarshalled),
	}

	err = p.output.WriteOne(ctx, msg)
	if err != nil {
		return err
	}

	return nil
}
