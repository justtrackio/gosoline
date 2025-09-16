package producer

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/stream"
	testEvent "github.com/justtrackio/gosoline/test/stream/kafka/test-event"
)

type producerModule struct {
	producer     stream.Producer
	produceCount int
}

func NewProducerModule(produceCount int, options ...stream.ProducerOption) kernel.ModuleFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
		producer, err := stream.NewProducer(ctx, config, logger, "testEvent", options...)
		if err != nil {
			return nil, fmt.Errorf("can not create producer: %w", err)
		}

		return &producerModule{
			producer:     producer,
			produceCount: produceCount,
		}, nil
	}
}

func (p producerModule) Run(ctx context.Context) error {
	for i := range p.produceCount {
		event := &testEvent.TestEvent{
			Id:   i,
			Name: fmt.Sprintf("event %d", i),
		}

		err := p.producer.WriteOne(ctx, event)
		if err != nil {
			return err
		}
	}

	return nil
}
