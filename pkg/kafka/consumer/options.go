package consumer

import (
	"github.com/segmentio/kafka-go"
)

const (
	// MaxBatchBytes is the Kafka default max batch size in bytes.
	MaxBatchBytes = 1000000

	// MaxBatchSize is the maximum batch size in number of messages.
	MaxBatchSize = 1024
)

type ReaderOption func(*kafka.ReaderConfig)

// WithBatch sets batching configuration.
func WithBatch(maxSize int) ReaderOption {
	return func(rc *kafka.ReaderConfig) {
		if maxSize > MaxBatchSize {
			maxSize = MaxBatchSize
		}

		rc.QueueCapacity = maxSize
	}
}

func getOptions(conf *Settings) []ReaderOption {
	opts := []ReaderOption{}

	if conf.BatchSize > 1 {
		opts = append(
			opts,
			WithBatch(conf.BatchSize),
		)
	}

	return opts
}
