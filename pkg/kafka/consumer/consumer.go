package consumer

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/coffin"
	"github.com/justtrackio/gosoline/pkg/kafka/connection"
	"github.com/justtrackio/gosoline/pkg/kafka/logging"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/segmentio/kafka-go"
)

type Offset struct {
	Partition int
	Index     int64
}

type Consumer struct {
	logger   log.Logger
	settings *Settings

	pool    coffin.Coffin
	backlog chan kafka.Message
	manager OffsetManager
}

func NewConsumer(
	ctx context.Context, conf cfg.Config, logger log.Logger, key string) (*Consumer, error) {
	settings := ParseSettings(conf, key)

	// Connection.
	dialer, err := connection.NewDialer(settings.Connection())
	if err != nil {
		return nil, fmt.Errorf("kafka: failed to get dialer: %w", err)
	}

	// Reader.
	reader, err := NewReader(logger, dialer, settings, getOptions(settings)...)
	if err != nil {
		return nil, fmt.Errorf("kafka: failed to get reader: %w", err)
	}

	manager := NewOffsetManager(logger, reader, settings.BatchSize, settings.BatchTimeout)

	return NewConsumerWithInterfaces(settings, logger, manager)
}

func NewConsumerWithInterfaces(settings *Settings, logger log.Logger, manager OffsetManager) (*Consumer, error) {
	logger = logger.WithFields(
		log.Fields{
			"kafka_topic":          settings.FQTopic,
			"kafka_consumer_group": settings.FQGroupID,
			"kafka_batch_size":     settings.BatchSize,
			"kafka_max_wait":       settings.BatchTimeout.Milliseconds(),
		},
	)

	return &Consumer{
		settings: settings,
		logger:   logging.NewKafkaLogger(logger),
		pool:     coffin.New(),
		backlog:  make(chan kafka.Message, settings.BatchSize),
		manager:  manager,
	}, nil
}

func (c *Consumer) Run(ctx context.Context) error {
	c.logger.Info("starting consumer")
	defer c.logger.Info("shutdown consumer")

	c.pool.GoWithContext(ctx, c.run)
	return c.pool.Wait()
}

func (c *Consumer) Data() chan kafka.Message {
	return c.backlog
}

func (c *Consumer) Commit(ctx context.Context, msgs ...kafka.Message) error {
	return c.manager.Commit(ctx, msgs...)
}

func (c *Consumer) run(ctx context.Context) error {
	c.pool.GoWithContext(ctx, c.manager.Start)

	for {
		for _, msg := range c.manager.Batch(ctx) {
			select {
			case c.backlog <- msg:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
}
