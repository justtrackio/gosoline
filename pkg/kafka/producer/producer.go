package producer

import (
	"context"
	"errors"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/coffin"
	"github.com/justtrackio/gosoline/pkg/kafka/connection"
	"github.com/justtrackio/gosoline/pkg/kafka/logging"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/segmentio/kafka-go"
)

var ErrInvalidMessage = errors.New("kafka: message is invalid")

type Producer struct {
	Settings *Settings
	Writer   Writer
	Logger   log.Logger
	pool     coffin.Coffin
}

// NewProducer returns a topic producer.
func NewProducer(ctx context.Context, conf cfg.Config, logger log.Logger, name string) (*Producer, error) {
	settings, err := ParseSettings(conf, name)
	if err != nil {
		return nil, fmt.Errorf("failed to parse kafka producer settings for %s: %w", name, err)
	}

	// Connection.
	dialer, err := connection.NewDialer(settings.Connection())
	if err != nil {
		return nil, fmt.Errorf("kafka: failed to get dialer: %w", err)
	}

	// Writer.
	writer, err := NewWriter(logger, dialer, settings.Connection().Bootstrap, getOptions(settings)...)
	if err != nil {
		return nil, fmt.Errorf("kafka: failed to get writer: %w", err)
	}

	return NewProducerWithInterfaces(settings, logger, writer)
}

func NewProducerWithInterfaces(conf *Settings, logger log.Logger, writer Writer) (*Producer, error) {
	logger = logger.WithFields(
		log.Fields{
			"kafka_topic":         conf.FQTopic,
			"kafka_batch_size":    conf.BatchSize,
			"kafka_batch_timeout": conf.BatchTimeout,
		},
	)

	return &Producer{
		Settings: conf,
		Writer:   writer,
		Logger:   logging.NewKafkaLogger(logger),
		pool:     coffin.New(),
	}, nil
}

// Run starts background routine for flushing messages.
func (p *Producer) Run(ctx context.Context) error {
	p.Logger.Info("starting producer")
	defer p.Logger.Info("shutdown producer")

	p.pool.GoWithContext(ctx, p.flushOnExit)

	return p.pool.Wait()
}

func (p *Producer) WriteOne(ctx context.Context, m kafka.Message) error {
	return p.write(ctx, m)
}

func (p *Producer) Write(ctx context.Context, ms ...kafka.Message) error {
	return p.write(ctx, ms...)
}

func (p *Producer) write(ctx context.Context, ms ...kafka.Message) error {
	ctx, cancel := context.WithTimeout(ctx, DefaultWriterWriteTimeout)
	defer cancel()

	p.Logger.Debug("producing messages")

	// Prepare batch.
	batch := []kafka.Message{}

	for _, m := range ms {
		batch = append(batch, kafka.Message{
			Topic:   p.Settings.FQTopic,
			Key:     m.Key,
			Value:   m.Value,
			Headers: m.Headers,
		})
	}

	return p.Writer.WriteMessages(ctx, batch...)
}

func (p *Producer) flushOnExit(ctx context.Context) error {
	<-ctx.Done()

	p.Logger.Info("flushing messages")
	defer p.Logger.Info("flushed messages")

	if err := p.Writer.Close(); err != nil {
		p.Logger.WithFields(log.Fields{"Error": err}).Error("failed to flush messages")
	}

	return nil
}
