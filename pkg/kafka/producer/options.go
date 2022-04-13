package producer

import (
	"time"

	"github.com/segmentio/kafka-go"
)

const (
	// KafkaDefaultBatchMessageCount is the Kafka default value for batch size.
	KafkaDefaultBatchMessageCount = 100

	// KafkaDefaultBatchMaxBytes is the Kafka default value for batch size.
	KafkaDefaultBatchMaxBytes = 1000012 - (100 * 1024)

	// KafkaMinBatchInterval is a reasonable minimum for batch timeout.
	KafkaMinBatchInterval = 250 * time.Millisecond
)

type WriterOption func(*kafka.WriterConfig)

// WithBatch sets batching configuration.
func WithBatch(maxCount int, interval time.Duration) WriterOption {
	return func(wc *kafka.WriterConfig) {
		if maxCount < 1 {
			maxCount = KafkaDefaultBatchMessageCount
		}

		if interval < KafkaMinBatchInterval {
			interval = KafkaMinBatchInterval
		}

		// Set maximum number of messages in a batch.
		wc.BatchSize = maxCount
		// Set flushing interval.
		wc.BatchTimeout = interval
		// Set maximum size of batch.
		wc.BatchBytes = KafkaDefaultBatchMaxBytes
	}
}

// WithAsyncWrites makes writes async.
func WithAsyncWrites() WriterOption {
	return func(wc *kafka.WriterConfig) {
		wc.Async = true
	}
}

func getOptions(conf *Settings) []WriterOption {
	opts := []WriterOption{}

	if conf.BatchSize > 1 || conf.BatchTimeout > time.Millisecond {
		opts = append(
			opts,
			WithBatch(conf.BatchSize, conf.BatchTimeout),
		)
	}
	if conf.AsyncWrites {
		opts = append(opts,
			WithAsyncWrites(),
		)
	}

	return opts
}
