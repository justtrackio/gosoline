package producer

import (
	"context"
	"time"

	"github.com/justtrackio/gosoline/pkg/kafka/logging"
	"github.com/justtrackio/gosoline/pkg/log"

	"github.com/segmentio/kafka-go"
)

const (
	// RequireAllReplicas means that ALL nodes in the replica-set must to confirm the write for a write
	// to be considered durable.
	RequireAllReplicas = -1

	// DefaultWriterWriteTimeout is how much to wait for a write to go through.
	DefaultWriterWriteTimeout = 30 * time.Second

	// DefaultWriterReadTimeout is how much to wait for reads.
	DefaultWriterReadTimeout = 30 * time.Second

	// DefaultMaxRetryAttempts is how many times to retry a failed operation.
	DefaultMaxRetryAttempts = 3

	// DefaultMetadataTTL is the frequency of metadata refreshes.
	DefaultMetadataTTL = 5 * time.Second

	// DefaultIdleTimeout is the period during which an idle connection can be resued.
	DefaultIdleTimeout = 30 * time.Second
)

//go:generate go run github.com/vektra/mockery/v2 --name Writer --unroll-variadic=False --with-expecter=False
type Writer interface {
	WriteMessages(ctx context.Context, msgs ...kafka.Message) error
	Stats() kafka.WriterStats
	Close() error
}

func NewWriter(
	logger log.Logger,
	dialer *kafka.Dialer,
	bootstrap []string,
	opts ...WriterOption,
) (*kafka.Writer, error) {
	// Config.
	conf := &kafka.WriterConfig{
		Brokers:  bootstrap,
		Balancer: &kafka.Hash{},
		Dialer:   dialer,

		// Non-batched by default.
		BatchSize: 1,
		Async:     false,

		// Use a safe default for durability.
		RequiredAcks: RequireAllReplicas,
		MaxAttempts:  DefaultMaxRetryAttempts,

		ReadTimeout:  DefaultWriterReadTimeout,
		WriteTimeout: DefaultWriterWriteTimeout,

		CompressionCodec: kafka.Snappy.Codec(),

		Logger:      logging.NewKafkaLogger(logger).DebugLogger(),
		ErrorLogger: logging.NewKafkaLogger(logger).ErrorLogger(),
	}
	for _, opt := range opts {
		opt(conf)
	}

	// Transport.
	transport := &kafka.Transport{
		Dial:        dialer.DialFunc,
		SASL:        dialer.SASLMechanism,
		TLS:         dialer.TLS,
		ClientID:    dialer.ClientID,
		DialTimeout: dialer.Timeout,
		IdleTimeout: DefaultIdleTimeout,

		MetadataTTL: DefaultMetadataTTL,
	}

	// Writer.
	writer := &kafka.Writer{
		Addr:      kafka.TCP(conf.Brokers...),
		Transport: transport,

		Topic: conf.Topic,

		ReadTimeout:  conf.ReadTimeout,
		WriteTimeout: conf.WriteTimeout,
		MaxAttempts:  conf.MaxAttempts,

		Async:        conf.Async,
		BatchSize:    conf.BatchSize,
		BatchBytes:   int64(conf.BatchBytes),
		BatchTimeout: conf.BatchTimeout,

		Balancer:     conf.Balancer,
		RequiredAcks: kafka.RequiredAcks(conf.RequiredAcks),

		Compression: kafka.Compression(conf.CompressionCodec.Code()),

		Logger:      conf.Logger,
		ErrorLogger: conf.ErrorLogger,
	}

	return writer, nil
}
