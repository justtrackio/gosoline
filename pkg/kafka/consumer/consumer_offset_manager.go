package consumer

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/segmentio/kafka-go"
)

var _ OffsetManager = &offsetManager{}

//go:generate go run github.com/vektra/mockery/v2 --name OffsetManager
type OffsetManager interface {
	Start(ctx context.Context) error
	Batch(ctx context.Context) []kafka.Message
	Commit(ctx context.Context, msgs ...kafka.Message) error
	Flush() error
	IsHealthy() bool
}

type offsetManager struct {
	logger               log.Logger
	reader               Reader
	readLock             *sync.Mutex
	incoming             chan kafka.Message
	batcher              Batcher
	uncomitted           map[Offset]int64
	uncomittedEmptyEvent chan bool
	healthCheckTimer     clock.HealthCheckTimer
	fetching             atomic.Bool
}

func NewOffsetManager(logger log.Logger, reader Reader, batchSize int, batchTimeout time.Duration, healthCheckTimer clock.HealthCheckTimer) *offsetManager {
	events := make(chan bool, 1)
	events <- true

	incoming := make(chan kafka.Message, batchSize)

	return &offsetManager{
		logger:               logger,
		reader:               reader,
		readLock:             &sync.Mutex{},
		incoming:             incoming,
		batcher:              NewBatcher(incoming, batchSize, batchTimeout),
		uncomitted:           map[Offset]int64{},
		uncomittedEmptyEvent: events,
		healthCheckTimer:     healthCheckTimer,
	}
}

func (m *offsetManager) Start(ctx context.Context) error {
	defer m.Flush()

	for {
		m.logger.Debug("fetching a message")

		// record we are fetching a message - while we are fetching, we can't get unhealthy
		// (as this code is outside our control to add code to mark us as healthy)
		m.fetching.Store(true)
		msg, err := m.reader.FetchMessage(ctx)
		// mark us as healthy as soon as we got a message to ensure we stay healthy while we process the message
		// (unless we take too long to send the message to the m.incoming channel)
		m.healthCheckTimer.MarkHealthy()
		m.fetching.Store(false)
		if err != nil {
			return err
		}

		m.logger.WithFields(log.Fields{
			"kafka_partition": msg.Partition,
			"kafka_offset":    msg.Offset,
			"kafka_key":       msg.Key,
		}).Debug("fetched a message")

		select {
		case m.incoming <- msg:
		case <-ctx.Done():
			close(m.incoming)

			return ctx.Err()
		}
	}
}

func (m *offsetManager) Batch(ctx context.Context) []kafka.Message {
	m.logger.Debug("compiling batch")
	defer m.logger.Debug("compiled batch")

	batch := []kafka.Message{}

	select {
	case <-m.uncomittedEmptyEvent:
		batch = m.batcher.Get(ctx)
	case <-ctx.Done():
		return batch
	}

	m.readLock.Lock()
	defer m.readLock.Unlock()

	batch = funk.Filter(batch, func(msg kafka.Message) bool {
		return !isControlMessage(msg)
	})

	if len(batch) == 0 {
		select {
		case m.uncomittedEmptyEvent <- true:
		default:
		}
	}

	for _, msg := range batch {
		m.uncomitted[Offset{Partition: msg.Partition, Index: msg.Offset}] = msg.Offset
	}

	return batch
}

func isControlMessage(msg kafka.Message) bool {
	// this is a control message indicating an aborted transactional message.
	// the kafka-go library does not support transactions currently and is not handling this correctly (https://github.com/segmentio/kafka-go/issues/1348).
	return len(msg.Value) == 6 && msg.Value[0] == 0 && msg.Value[1] == 0 && msg.Value[2] == 0 && msg.Value[3] == 0
}

func (m *offsetManager) Commit(ctx context.Context, msgs ...kafka.Message) error {
	logger := m.logger.WithFields(log.Fields{
		"kafka_batch_size": len(msgs),
	})

	logger.Debug("committing offsets")
	defer logger.Debug("committed offsets")

	m.readLock.Lock()
	defer m.readLock.Unlock()
	for _, msg := range msgs {
		key := Offset{Partition: msg.Partition, Index: msg.Offset}
		if _, exists := m.uncomitted[key]; !exists {
			m.logger.WithFields(log.Fields{
				"kafka_partition": msg.Partition,
				"kafka_offset":    msg.Offset,
				"kafka_key":       msg.Key,
				"Error":           "commit unknown message",
			}).Error("failed to commit message")
		}

		delete(m.uncomitted, key)
	}
	// Kafka is a stream and sequential in nature, there are no per-message acks/nacks,
	// instead acks are done through offsets (similar to TCP sequence numbers), in other words,
	// committing an offset implies committing everything that came before it,
	// as such a batch must be committed before another one can be requested.
	if len(m.uncomitted) == 0 {
		m.uncomittedEmptyEvent <- true
	}

	return m.reader.CommitMessages(ctx, msgs...)
}

func (m *offsetManager) Flush() error {
	m.logger.Info("flushing messages")
	defer m.logger.Info("flushed messages")

	if err := m.reader.Close(); err != nil {
		m.logger.WithFields(log.Fields{"Error": err}).Error("failed to flush messages")

		return err
	}

	return nil
}

func (m *offsetManager) IsHealthy() bool {
	return m.healthCheckTimer.IsHealthy() || m.fetching.Load()
}
