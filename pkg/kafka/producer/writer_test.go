package producer_test

import (
	"crypto/tls"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/kafka/connection"
	"github.com/justtrackio/gosoline/pkg/kafka/producer"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
)

var (
	writerDialer = &kafka.Dialer{
		TLS: &tls.Config{
			InsecureSkipVerify: true,
		},
		ClientID:      "my-client",
		SASLMechanism: kafka.DefaultDialer.SASLMechanism,
	}

	writerConf = (&producer.Settings{
		Topic:   "my-topic",
		FQTopic: "test-my-topic",
	}).WithConnection(&connection.Settings{
		Bootstrap: []string{"kafka.domain.tld:9094"},
	})
)

func TestTransport(t *testing.T) {
	writer, err := producer.NewWriter(
		logMocks.NewLoggerMockedAll(), writerDialer, writerConf.Connection().Bootstrap,
		producer.WithAsyncWrites(),
	)
	assert.Nil(t, err)

	assert.Equal(t, writer.Addr, kafka.TCP(writerConf.Connection().Bootstrap...))
	assert.Equal(t, writer.Balancer, &kafka.Hash{})

	transport := writer.Transport.(*kafka.Transport)

	assert.Equal(t, &transport.Dial, &writerDialer.DialFunc)
	assert.Equal(t, transport.SASL, writerDialer.SASLMechanism)
	assert.Equal(t, transport.TLS, writerDialer.TLS)
	assert.Equal(t, transport.DialTimeout, writerDialer.Timeout)
	// High values for idle timeout is known to cause an issue, where
	// broker closes the connection and client unknowingly keeps using it.
	assert.Equal(t, transport.IdleTimeout, 30*time.Second)
	assert.Equal(t, transport.MetadataTTL, 5*time.Second)
}

func TestSaneDefaults(t *testing.T) {
	writer, err := producer.NewWriter(logMocks.NewLoggerMockedAll(), writerDialer, writerConf.Connection().Bootstrap)
	assert.Nil(t, err)

	// Endpoint
	assert.Equal(t, writer.Addr.String(), writerConf.Connection().Bootstrap[0])

	// Safetyß
	assert.Equal(t, int(writer.RequiredAcks), -1)
	assert.Equal(t, writer.MaxAttempts, 3)
	assert.Equal(t, writer.WriteTimeout, 30*time.Second)

	// Non-batched by default.
	assert.Equal(t, writer.BatchSize, 1)
	assert.Equal(t, writer.Async, false)

	// Performance.
	assert.Equal(t, writer.Compression, kafka.Snappy)
}

func TestSaneDefaultsFallback(t *testing.T) {
	writer, err := producer.NewWriter(
		logMocks.NewLoggerMockedAll(), writerDialer, writerConf.Connection().Bootstrap,
		producer.WithBatch(0, time.Microsecond),
	)
	assert.Nil(t, err)

	assert.Equal(t, writer.BatchSize, producer.KafkaDefaultBatchMessageCount)
	assert.Equal(t, writer.BatchTimeout, producer.KafkaMinBatchInterval)
}

func TestWithBatch(t *testing.T) {
	const (
		batchSize     = 50
		batchInterval = time.Second
	)

	writer, err := producer.NewWriter(
		logMocks.NewLoggerMockedAll(), writerDialer, writerConf.Connection().Bootstrap,
		producer.WithBatch(batchSize, batchInterval),
	)
	assert.Nil(t, err)

	assert.Equal(t, writer.BatchSize, batchSize)
	assert.Equal(t, writer.BatchTimeout, batchInterval)
	assert.False(t, writer.Async)
}

func TestWithAsyncWrites(t *testing.T) {
	writer, err := producer.NewWriter(
		logMocks.NewLoggerMockedAll(), writerDialer, writerConf.Connection().Bootstrap,
		producer.WithAsyncWrites(),
	)

	assert.Nil(t, err)
	assert.True(t, writer.Async)
}
