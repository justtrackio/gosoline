package consumer_test

import (
	"context"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/justtrackio/gosoline/pkg/kafka/consumer"
	kafkaConsumerMocks "github.com/justtrackio/gosoline/pkg/kafka/consumer/mocks"
	"github.com/justtrackio/gosoline/pkg/log"
	metricMocks "github.com/justtrackio/gosoline/pkg/metric/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/twmb/franz-go/pkg/kerr"
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
