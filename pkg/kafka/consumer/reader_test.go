package consumer_test

import (
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/kafka/connection"
	"github.com/justtrackio/gosoline/pkg/kafka/consumer"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
)

var (
	readerDialer   = &kafka.Dialer{ClientID: "my-client"}
	readerSettings = (&consumer.Settings{
		Topic:     "my-topic",
		FQTopic:   "test-my-topic",
		FQGroupID: "my-group",
	}).WithConnection(&connection.Settings{
		Bootstrap: []string{"kafka.domain.tld:9094"},
	})
)

func TestSaneDefaults(t *testing.T) {
	reader, err := consumer.NewReader(logMocks.NewLoggerMockedAll(), readerDialer, readerSettings)
	assert.Nil(t, err)

	assert.Equal(t, reader.Config().MaxAttempts, 3)

	assert.Equal(t, reader.Config().MaxBytes, 1000000)
	assert.Equal(t, reader.Config().CommitInterval, time.Duration(0))
	assert.Equal(t, reader.Config().RetentionTime, time.Hour*24*7)

	assert.Equal(t, reader.Config().Brokers, readerSettings.Connection().Bootstrap)

	assert.Equal(t, reader.Config().Topic, readerSettings.FQTopic)
	assert.Equal(t, reader.Config().GroupID, readerSettings.FQGroupID)
}

func TestFallsbackToSaneDefaults(t *testing.T) {
	reader, err := consumer.NewReader(
		logMocks.NewLoggerMockedAll(),
		readerDialer,
		readerSettings,
		consumer.WithBatch(1e6),
	)
	assert.Nil(t, err)

	assert.Equal(t, reader.Config().QueueCapacity, 1024)
	assert.Equal(t, reader.Config().MaxWait, time.Second)
}

func TestAppliesWithBatch(t *testing.T) {
	const (
		batchMaxSize = 50
	)

	reader, err := consumer.NewReader(
		logMocks.NewLoggerMockedAll(),
		readerDialer,
		readerSettings,
		consumer.WithBatch(batchMaxSize),
	)
	assert.Nil(t, err)
	assert.Equal(t, reader.Config().QueueCapacity, batchMaxSize)
}
