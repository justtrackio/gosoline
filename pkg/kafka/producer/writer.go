package producer

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kafka"
	"github.com/justtrackio/gosoline/pkg/kafka/connection"
	"github.com/justtrackio/gosoline/pkg/kafka/logging"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/reslife"
	"github.com/twmb/franz-go/pkg/kgo"
)

//go:generate go run github.com/vektra/mockery/v2 --name Writer
type Writer interface {
	ProduceSync(ctx context.Context, rs ...*kgo.Record) kgo.ProduceResults
}

func NewWriter(ctx context.Context, config cfg.Config, logger log.Logger, settings *Settings) (Writer, error) {
	if err := settings.PadFromConfig(config); err != nil {
		return nil, fmt.Errorf("failed to pad app id from config: %w", err)
	}

	topic, err := kafka.BuildFullTopicName(config, settings.AppId, settings.TopicId)
	if err != nil {
		return nil, fmt.Errorf("failed to build full topic name for topic id %q: %w", settings.TopicId, err)
	}

	opts := []kgo.Opt{
		kgo.DefaultProduceTopic(topic),
		kgo.ProducerBatchMaxBytes(settings.MaxBatchBytes),
		kgo.MaxBufferedRecords(settings.MaxBatchSize),
		kgo.ProducerLinger(settings.LingerTimeout),
		kgo.ProduceRequestTimeout(settings.RequestTimeout),
		kgo.ProducerBatchCompression(settings.GetKafkaCompressor()),
		kgo.WithContext(ctx),
		kgo.WithLogger(logging.NewKafkaLogger(ctx, logger)),
	}

	connOpts, err := connection.BuildConnectionOptions(config, settings.Connection)
	if err != nil {
		return nil, fmt.Errorf("failed to build connection options: %w", err)
	}
	opts = append(opts, connOpts...)

	client, err := kgo.NewClient(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create franz-go client: %w", err)
	}

	if err = reslife.AddLifeCycleer(ctx, kafka.NewLifecycleManager(settings.Connection, topic)); err != nil {
		return nil, fmt.Errorf("failed to add kafka lifecycle manager: %w", err)
	}

	return client, nil
}
