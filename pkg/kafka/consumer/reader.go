package consumer

import (
	"context"
	"time"

	"github.com/justtrackio/gosoline/pkg/kafka/logging"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/segmentio/kafka-go"
)

const (
	// DefaultMaxRetryAttempts is how many times to retry a failed operation.
	DefaultMaxRetryAttempts = 3

	// DefaultConsumerGroupRetentionTime is the retention period of current offsets.
	DefaultConsumerGroupRetentionTime = 7 * 24 * time.Hour

	// DefaultMaxWait is a reasonable minimum for MaxWait.
	DefaultMaxWait = time.Second

	// CommitOffsetsSync == 0 means that auto-commit is disabled.
	CommitOffsetsSync = time.Duration(0)
)

//go:generate go run github.com/vektra/mockery/v2 --name Reader
type Reader interface {
	FetchMessage(context.Context) (kafka.Message, error)
	ReadMessage(context.Context) (kafka.Message, error)
	CommitMessages(context.Context, ...kafka.Message) error
	Stats() kafka.ReaderStats
	Close() error
}

func NewReader(
	logger log.Logger,
	dialer *kafka.Dialer,
	settings *Settings,
	opts ...ReaderOption,
) (*kafka.Reader, error) {
	startOffset := kafka.FirstOffset
	if settings.StartOffset == "last" {
		startOffset = kafka.LastOffset
	}

	c := &kafka.ReaderConfig{
		Brokers: settings.Connection().Bootstrap,
		Dialer:  dialer,

		// Topic.
		Topic:                 settings.FQTopic,
		GroupID:               settings.FQGroupID,
		WatchPartitionChanges: true,

		// No batching by default.
		MinBytes: 1,
		MaxBytes: MaxBatchBytes,
		MaxWait:  DefaultMaxWait,

		// Explicit commits.
		CommitInterval: CommitOffsetsSync,

		// Safe defaults.
		RetentionTime:  DefaultConsumerGroupRetentionTime,
		MaxAttempts:    DefaultMaxRetryAttempts,
		IsolationLevel: kafka.ReadCommitted,

		StartOffset: startOffset,

		Logger:      logging.NewKafkaLogger(logger).DebugLogger(),
		ErrorLogger: logging.NewKafkaLogger(logger).ErrorLogger(),
	}

	for _, opt := range opts {
		opt(c)
	}

	if err := c.Validate(); err != nil {
		return nil, err
	}

	return kafka.NewReader(*c), nil
}
