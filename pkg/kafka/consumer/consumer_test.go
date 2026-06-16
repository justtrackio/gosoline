package consumer_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/justtrackio/gosoline/pkg/kafka"
	"github.com/justtrackio/gosoline/pkg/kafka/consumer"
	kafkaConsumerMocks "github.com/justtrackio/gosoline/pkg/kafka/consumer/mocks"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/metric"
	metricMocks "github.com/justtrackio/gosoline/pkg/metric/mocks"
	"github.com/justtrackio/gosoline/pkg/test/matcher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kgo"
	"golang.org/x/sys/unix"
)

func TestConsumerRunStopReturns(t *testing.T) {
	reader := kafkaConsumerMocks.NewReader(t)
	handler := kafkaConsumerMocks.NewKafkaMessageHandler(t)
	metricWriter := metricMocks.NewWriter(t)

	reader.EXPECT().CloseAllowingRebalance().Once()
	handler.EXPECT().Stop().Once()

	readerFactory := func(_ context.Context, _ *consumer.PartitionManager) (consumer.Reader, error) {
		return reader, nil
	}

	healthCheckTimer := clock.NewHealthCheckTimerWithInterfaces(clock.NewFakeClock(), time.Minute)

	c := consumer.NewConsumerWithInterfaces(
		log.NewLogger(),
		clock.NewFakeClock(),
		healthCheckTimer,
		handler,
		readerFactory,
		consumer.Settings{
			MaxPollRecords: 100,
			IdleWaitTime:   500 * time.Millisecond,
		},
		metricWriter,
		"test-topic",
		false,
		"test-consumer",
	)

	ctx := context.Background()
	c.Stop(ctx)

	err := c.Run(ctx)
	assert.NoError(t, err)
}

func TestConsumerIsHealthy(t *testing.T) {
	handler := kafkaConsumerMocks.NewKafkaMessageHandler(t)
	metricWriter := metricMocks.NewWriter(t)
	fakeClock := clock.NewFakeClock()
	healthCheckTimer := clock.NewHealthCheckTimerWithInterfaces(fakeClock, time.Minute)

	readerFactory := func(_ context.Context, _ *consumer.PartitionManager) (consumer.Reader, error) {
		return nil, nil
	}

	c := consumer.NewConsumerWithInterfaces(
		log.NewLogger(),
		fakeClock,
		healthCheckTimer,
		handler,
		readerFactory,
		consumer.Settings{
			MaxPollRecords: 100,
			IdleWaitTime:   500 * time.Millisecond,
		},
		metricWriter,
		"test-topic",
		false,
		"test-consumer",
	)

	// Initially healthy (timer hasn't expired)
	assert.True(t, c.IsHealthy())

	// Advance past the timeout
	fakeClock.Advance(2 * time.Minute)

	// Should be unhealthy after timeout
	assert.False(t, c.IsHealthy())
}

func TestCheckKafkaRetryableError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected exec.ErrorType
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: exec.ErrorTypeOk,
		},
		{
			name:     "connection refused",
			err:      unix.ECONNREFUSED,
			expected: exec.ErrorTypeRetryable,
		},
		{
			name:     "connection reset",
			err:      unix.ECONNRESET,
			expected: exec.ErrorTypeRetryable,
		},
		{
			name:     "EOF",
			err:      io.EOF,
			expected: exec.ErrorTypeRetryable,
		},
		{
			name:     "unexpected EOF",
			err:      io.ErrUnexpectedEOF,
			expected: exec.ErrorTypeRetryable,
		},
		{
			name:     "broken pipe",
			err:      unix.EPIPE,
			expected: exec.ErrorTypeRetryable,
		},
		{
			name:     "wrapped connection refused",
			err:      errors.Join(errors.New("unable to dial"), unix.ECONNREFUSED),
			expected: exec.ErrorTypeRetryable,
		},
		{
			name:     "kafka retryable error - NotLeaderForPartition (code 6)",
			err:      kerr.ErrorForCode(6),
			expected: exec.ErrorTypeRetryable,
		},
		{
			name:     "kafka retryable error - LeaderNotAvailable (code 5)",
			err:      kerr.ErrorForCode(5),
			expected: exec.ErrorTypeRetryable,
		},
		{
			name:     "kafka non-retryable error - RebalanceInProgress (code 27)",
			err:      kerr.ErrorForCode(27),
			expected: exec.ErrorTypeUnknown,
		},
		{
			name:     "kafka non-retryable error - InvalidTopic (code 17)",
			err:      kerr.ErrorForCode(17),
			expected: exec.ErrorTypeUnknown,
		},
		{
			name:     "kafka retryable error - UnknownTopicOrPartition (code 3)",
			err:      kerr.ErrorForCode(3),
			expected: exec.ErrorTypeRetryable,
		},
		{
			name:     "generic error",
			err:      errors.New("some random error"),
			expected: exec.ErrorTypeUnknown,
		},
		{
			name:     "net dial error with connection refused",
			err:      &net.OpError{Op: "dial", Err: unix.ECONNREFUSED},
			expected: exec.ErrorTypeRetryable,
		},
		{
			name:     "no such host",
			err:      &net.DNSError{Err: "no such host", IsNotFound: true},
			expected: exec.ErrorTypeRetryable,
		},
		{
			name:     "no route to host",
			err:      unix.EHOSTUNREACH,
			expected: exec.ErrorTypeRetryable,
		},
		{
			name:     "network unreachable",
			err:      unix.ENETUNREACH,
			expected: exec.ErrorTypeRetryable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kafkaReader := kafkaConsumerMocks.NewReader(t)

			if tt.expected == exec.ErrorTypeRetryable {
				kafkaReader.EXPECT().AllowRebalance().Once()
			}

			result := consumer.CheckKafkaRetryableError(kafkaReader)(nil, tt.err)
			assert.Equal(t, tt.expected, result, "CheckKafkaRetryableError(nil, %v) = %v, want %v", tt.err, result, tt.expected)
		})
	}
}

func TestCheckKafkaUnknownTopicError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected exec.ErrorType
	}{
		{
			name:     "nil error falls through",
			err:      nil,
			expected: exec.ErrorTypeUnknown,
		},
		{
			name:     "UnknownTopicOrPartition is permanent",
			err:      kerr.UnknownTopicOrPartition,
			expected: exec.ErrorTypePermanent,
		},
		{
			name:     "UnknownTopicID is permanent",
			err:      kerr.UnknownTopicID,
			expected: exec.ErrorTypePermanent,
		},
		{
			name:     "wrapped UnknownTopicOrPartition is permanent",
			err:      fmt.Errorf("failed to fetch records (topic: %s, partition: %d): %w", "some-topic", 0, kerr.UnknownTopicOrPartition),
			expected: exec.ErrorTypePermanent,
		},
		{
			name: "joined wrapped UnknownTopicOrPartition is permanent",
			err: errors.Join(
				fmt.Errorf("failed to fetch records (topic: %s, partition: %d): %w", "other-topic", 1, kerr.LeaderNotAvailable),
				fmt.Errorf("failed to fetch records (topic: %s, partition: %d): %w", "some-topic", 0, kerr.UnknownTopicOrPartition),
			),
			expected: exec.ErrorTypePermanent,
		},
		{
			name:     "retryable kafka error falls through",
			err:      kerr.NotLeaderForPartition,
			expected: exec.ErrorTypeUnknown,
		},
		{
			name:     "generic error falls through",
			err:      errors.New("some random error"),
			expected: exec.ErrorTypeUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := consumer.CheckKafkaUnknownTopicError(nil, tt.err)
			assert.Equal(t, tt.expected, result, "CheckKafkaUnknownTopicError(nil, %v) = %v, want %v", tt.err, result, tt.expected)
		})
	}
}

// A missing topic must NOT be ignored; it has to be surfaced so the executor fails the consumer fast.
// franz-go surfaces this as UNKNOWN_TOPIC_OR_PARTITION or UNKNOWN_TOPIC_ID once KeepRetryableFetchErrors
// is enabled.
func TestConsumerRunFailsFastOnUnknownTopic(t *testing.T) {
	for name, partitionErr := range map[string]error{
		"UnknownTopicOrPartition": kerr.UnknownTopicOrPartition,
		"UnknownTopicID":          kerr.UnknownTopicID,
	} {
		t.Run(name, func(t *testing.T) {
			reader := kafkaConsumerMocks.NewReader(t)
			handler := kafkaConsumerMocks.NewKafkaMessageHandler(t)
			metricWriter := metricMocks.NewWriter(t)

			reader.EXPECT().PollRecords(nil, 100).Return(fetchWithPartitionError(nil, partitionErr)).Once()
			reader.EXPECT().CloseAllowingRebalance().Once()
			handler.EXPECT().Stop().Once()

			readerFactory := func(_ context.Context, _ *consumer.PartitionManager) (consumer.Reader, error) {
				return reader, nil
			}

			fakeClock := clock.NewFakeClock()
			healthCheckTimer := clock.NewHealthCheckTimerWithInterfaces(fakeClock, time.Minute)

			c := consumer.NewConsumerWithInterfaces(
				log.NewLogger(),
				fakeClock,
				healthCheckTimer,
				handler,
				readerFactory,
				consumer.Settings{
					MaxPollRecords: 100,
					IdleWaitTime:   500 * time.Millisecond,
				},
				metricWriter,
				"missing-topic",
				false,
				"test-consumer",
			)

			err := c.Run(t.Context())
			require.Error(t, err)
			assert.True(t, errors.Is(err, partitionErr), "expected error to wrap %v, got: %v", partitionErr, err)
		})
	}
}

// A retryable, non-unknown-topic fetch error (e.g. NOT_LEADER_FOR_PARTITION) must be ignored so that the
// records delivered alongside it are still processed and the consumer keeps running.
func TestConsumerRunIgnoresRetryableFetchError(t *testing.T) {
	reader := kafkaConsumerMocks.NewReader(t)
	handler := kafkaConsumerMocks.NewKafkaMessageHandler(t)
	metricWriter := metricMocks.NewWriter(t)

	records := []*kgo.Record{{Value: []byte("payload")}}
	// The first poll returns a record alongside a retryable per-partition error; the second poll reports the
	// client as closed so Run leaves its loop.
	reader.EXPECT().PollRecords(nil, 100).Return(fetchWithPartitionError(records, kerr.NotLeaderForPartition)).Once()
	reader.EXPECT().PollRecords(nil, 100).Return(clientClosedFetches()).Once()
	reader.EXPECT().AllowRebalance().Once()
	reader.EXPECT().CloseAllowingRebalance().Once()

	handler.EXPECT().Handle(records).Once()
	handler.EXPECT().Stop().Once()

	dims := metric.Dimensions{
		kafka.DimensionClientType: kafka.DimensionConsumer,
		kafka.DimensionClient:     "test-consumer",
		kafka.DimensionTopic:      "test-topic",
	}
	expectedMetrics := metric.Data{
		metric.NewMetricDatum("PollCount", dims, 1.0, metric.UnitCount, metric.PriorityHigh),
		metric.NewMetricDatum("PollDuration", dims, 0.0, metric.UnitMillisecondsAverage, metric.PriorityHigh),
		metric.NewMetricDatum("RecordsConsumed", dims, 1.0, metric.UnitCount, metric.PriorityHigh),
	}
	metricWriter.EXPECT().Write(matcher.Context, expectedMetrics).Once()

	readerFactory := func(_ context.Context, _ *consumer.PartitionManager) (consumer.Reader, error) {
		return reader, nil
	}

	fakeClock := clock.NewFakeClock()
	healthCheckTimer := clock.NewHealthCheckTimerWithInterfaces(fakeClock, time.Minute)

	c := consumer.NewConsumerWithInterfaces(
		log.NewLogger(),
		fakeClock,
		healthCheckTimer,
		handler,
		readerFactory,
		consumer.Settings{
			MaxPollRecords: 100,
			IdleWaitTime:   500 * time.Millisecond,
		},
		metricWriter,
		"test-topic",
		true,
		"test-consumer",
	)

	err := c.Run(t.Context())
	assert.NoError(t, err, "a retryable non-unknown-topic fetch error should be ignored")
}

// A non-retryable fetch error must be surfaced so the executor treats it as permanent and fails the consumer.
func TestConsumerRunSurfacesNonRetryableFetchError(t *testing.T) {
	reader := kafkaConsumerMocks.NewReader(t)
	handler := kafkaConsumerMocks.NewKafkaMessageHandler(t)
	metricWriter := metricMocks.NewWriter(t)

	// InvalidTopicException (code 17) is not retryable.
	reader.EXPECT().PollRecords(nil, 100).Return(fetchWithPartitionError(nil, kerr.InvalidTopicException)).Once()
	reader.EXPECT().CloseAllowingRebalance().Once()
	handler.EXPECT().Stop().Once()

	readerFactory := func(_ context.Context, _ *consumer.PartitionManager) (consumer.Reader, error) {
		return reader, nil
	}

	fakeClock := clock.NewFakeClock()
	healthCheckTimer := clock.NewHealthCheckTimerWithInterfaces(fakeClock, time.Minute)

	c := consumer.NewConsumerWithInterfaces(
		log.NewLogger(),
		fakeClock,
		healthCheckTimer,
		handler,
		readerFactory,
		consumer.Settings{
			MaxPollRecords: 100,
			IdleWaitTime:   500 * time.Millisecond,
		},
		metricWriter,
		"test-topic",
		false,
		"test-consumer",
	)

	err := c.Run(t.Context())
	require.Error(t, err)
	assert.True(t, errors.Is(err, kerr.InvalidTopicException), "expected error to wrap INVALID_TOPIC_EXCEPTION, got: %v", err)
}

// fetchWithPartitionError builds a single-partition Fetches carrying the given records alongside a
// per-partition fetch error. franz-go may surface per-partition errors this way once
// KeepRetryableFetchErrors is enabled.
func fetchWithPartitionError(records []*kgo.Record, partitionErr error) kgo.Fetches {
	return kgo.Fetches{
		{
			Topics: []kgo.FetchTopic{
				{
					Topic: "test-topic",
					Partitions: []kgo.FetchPartition{
						{Partition: 0, Records: records, Err: partitionErr},
					},
				},
			},
		},
	}
}

func clientClosedFetches() kgo.Fetches {
	return kgo.Fetches{
		{
			Topics: []kgo.FetchTopic{
				{
					Topic: "test-topic",
					Partitions: []kgo.FetchPartition{
						{Partition: 0, Err: kgo.ErrClientClosed},
					},
				},
			},
		},
	}
}
