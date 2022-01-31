package consumer

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/mdlsub"
	"github.com/justtrackio/gosoline/pkg/stream"
)

func NewConsumer() stream.ConsumerCallbackFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (stream.ConsumerCallback, error) {
		publisher, err := mdlsub.NewPublisher(ctx, config, logger, "outputEvent")
		if err != nil {
			return nil, fmt.Errorf("can not create publisher: %w", err)
		}

		return &Consumer{
			publisher: publisher,
		}, nil
	}
}

type Consumer struct {
	publisher mdlsub.Publisher
}

func (c *Consumer) GetModel(map[string]interface{}) interface{} {
	return mdl.Uint(0)
}

func (c *Consumer) Consume(ctx context.Context, model interface{}, _ map[string]interface{}) (bool, error) {
	input := model.(*uint)
	*input++

	err := c.publisher.Publish(ctx, mdlsub.TypeCreate, 0, input, map[string]interface{}{})
	if err != nil {
		return false, fmt.Errorf("can not publish event: %w", err)
	}

	return true, nil
}
