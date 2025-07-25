package producer

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/kafka/logging"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/twmb/franz-go/pkg/kgo"
)

//go:generate go run github.com/vektra/mockery/v2 --name Writer
type Writer interface {
	ProduceSync(ctx context.Context, rs ...*kgo.Record) kgo.ProduceResults
}

func NewWriter(ctx context.Context, logger log.Logger, settings *Settings, options ...kgo.Opt) (Writer, error) {
	opts := []kgo.Opt{
		kgo.DefaultProduceTopic(settings.Topic),
		kgo.ProducerBatchCompression(settings.GetKafkaCompressor()),
		kgo.SeedBrokers(settings.Connection.Bootstrap...),
		kgo.WithContext(ctx),
		kgo.WithLogger(logging.NewKafkaLogger(logger)),
	}

	// todo: add TLS/Dialer configuration
	// todo: add partitioner

	opts = append(opts, options...)

	client, err := kgo.NewClient(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create franz-go client: %w", err)
	}

	return client, nil
}
