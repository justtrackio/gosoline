package consumer

import (
	"context"
	"sync"
	"time"

	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/segmentio/kafka-go"
)

var _ OffsetManager = &offsetManager{}

//go:generate mockery --name OffsetManager
type OffsetManager interface {
	Start(ctx context.Context) error
	Batch(ctx context.Context) []kafka.Message
	Commit(ctx context.Context, msgs ...kafka.Message) error
	Flush() error
}

type offsetManager struct {
	logger               log.Logger
	reader               Reader
	readLock             *sync.Mutex
	incoming             chan kafka.Message
	batcher              Batcher
	uncomitted           map[Offset]int64
	uncomittedEmptyEvent chan bool
}

func NewOffsetManager(logger log.Logger, reader Reader, batchSize int, batchTimeout time.Duration) *offsetManager {
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
	}
}

func (m *offsetManager) Start(ctx context.Context) error {
	defer m.Flush()

	for {
		m.logger.Debug("fetching a message")

		msg, err := m.reader.FetchMessage(ctx)
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
	for _, msg := range batch {
		m.uncomitted[Offset{Partition: msg.Partition, Index: msg.Offset}] = msg.Offset
	}

	return batch
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
