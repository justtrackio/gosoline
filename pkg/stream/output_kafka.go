package stream

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/coffin"
	kafkaProducer "github.com/justtrackio/gosoline/pkg/kafka/producer"
	"github.com/justtrackio/gosoline/pkg/log"
)

type KafkaOutput struct {
	producer *kafkaProducer.Producer
	pool     coffin.Coffin
}

var _ Output = &KafkaOutput{}

func NewKafkaOutput(ctx context.Context, config cfg.Config, logger log.Logger, key string) (*KafkaOutput, error) {
	prod, err := kafkaProducer.NewProducer(ctx, config, logger, key)
	if err != nil {
		return nil, fmt.Errorf("failed to init producer: %w", err)
	}

	return NewKafkaOutputWithInterfaces(ctx, prod)
}

func NewKafkaOutputWithInterfaces(ctx context.Context, producer *kafkaProducer.Producer) (*KafkaOutput, error) {
	pool := coffin.New()
	pool.GoWithContext(ctx, producer.Run)

	return &KafkaOutput{producer: producer, pool: pool}, nil
}

func (o *KafkaOutput) WriteOne(ctx context.Context, m WritableMessage) error {
	return o.producer.WriteOne(ctx, NewKafkaMessage(m))
}

func (o *KafkaOutput) Write(ctx context.Context, ms []WritableMessage) error {
	return o.producer.Write(ctx, NewKafkaMessages(ms)...)
}
