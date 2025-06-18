package consumertest

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/stream"
)

type Todo struct {
	Id     int
	Text   string
	Status string
}

type Consumer struct {
	producer stream.Producer
}

func NewConsumer(ctx context.Context, config cfg.Config, logger log.Logger) (stream.ConsumerCallback[Todo], error) {
	var err error
	var producer stream.Producer

	if producer, err = stream.NewProducer(ctx, config, logger, "todos"); err != nil {
		return nil, fmt.Errorf("can not create the todos producer: %w", err)
	}

	consumer := &Consumer{
		producer: producer,
	}

	return consumer, nil
}

func (c Consumer) Consume(ctx context.Context, todo Todo, attributes map[string]string) (bool, error) {
	todo.Status = "pending"

	if err := c.producer.WriteOne(ctx, todo); err != nil {
		return false, fmt.Errorf("can not write todo with id %d: %w", todo.Id, err)
	}

	return true, nil
}
