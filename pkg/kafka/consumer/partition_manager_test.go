package consumer_test

import (
	"context"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/kafka/consumer"
	kafkaConsumerMocks "github.com/justtrackio/gosoline/pkg/kafka/consumer/mocks"
	"github.com/justtrackio/gosoline/pkg/log"
	metricMocks "github.com/justtrackio/gosoline/pkg/metric/mocks"
	"github.com/justtrackio/gosoline/pkg/stream/health"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"github.com/twmb/franz-go/pkg/kgo"
)

func TestPartitionManagerTestSuite(t *testing.T) {
	suite.Run(t, new(PartitionManagerTestSuite))
}

type PartitionManagerTestSuite struct {
	suite.Suite
}

func (s *PartitionManagerTestSuite) TestDelayConsumeDisabled() {
	clk := clock.NewFakeClock()
	records := []*kgo.Record{{Timestamp: clk.Now()}}
	c := s.newConsumeDelayTestConsumer(clk, records, 0, time.Minute)

	s.NoError(c.Run(s.T().Context()))
}

func (s *PartitionManagerTestSuite) TestDelayConsumeWaitsForNewestRecordInBatch() {
	clk := clock.NewFakeClock()
	records := []*kgo.Record{
		{Timestamp: clk.Now().Add(-time.Minute)},
		{Timestamp: clk.Now()},
	}
	c := s.newConsumeDelayTestConsumer(clk, records, time.Second, time.Minute)
	result := make(chan error, 1)

	go func() {
		result <- c.Run(s.T().Context())
	}()

	clk.BlockUntilTimers(1)
	clk.Advance(time.Second)

	s.NoError(<-result)
}

func (s *PartitionManagerTestSuite) TestDelayConsumeSkipsOldBatch() {
	clk := clock.NewFakeClock()
	clk.Advance(time.Minute)
	records := []*kgo.Record{{Timestamp: clk.Now().Add(-time.Minute)}}
	c := s.newConsumeDelayTestConsumer(clk, records, time.Second, time.Minute)

	s.NoError(c.Run(s.T().Context()))
}

func (s *PartitionManagerTestSuite) TestDelayConsumeStopsOnCancellation() {
	clk := clock.NewFakeClock()
	records := []*kgo.Record{{Timestamp: clk.Now()}}
	c := s.newCanceledConsumeDelayTestConsumer(clk, records, time.Minute)
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)

	go func() {
		result <- c.Run(ctx)
	}()

	clk.BlockUntilTimers(1)
	cancel()

	s.ErrorIs(<-result, context.Canceled)
}

func (s *PartitionManagerTestSuite) TestDelayConsumeKeepsHealthCheckHealthy() {
	clk := clock.NewFakeClock()
	healthCheck := clock.NewHealthCheckTimerWithInterfaces(clk, time.Minute)
	records := []*kgo.Record{{Timestamp: clk.Now()}}
	c := s.newConsumeDelayTestConsumerWithHealthCheck(clk, healthCheck, records, 2*time.Minute, time.Minute)
	result := make(chan error, 1)

	go func() {
		result <- c.Run(s.T().Context())
	}()

	clk.BlockUntilTimers(1)
	clk.BlockUntilTickers(1)
	clk.Advance(time.Minute)

	s.True(healthCheck.IsHealthy())

	clk.Advance(time.Minute)
	s.Require().NoError(<-result)
}

func (s *PartitionManagerTestSuite) TestDelayConsumeSupportsMinimumHealthTimeout() {
	clk := clock.NewFakeClock()
	healthCheck := clock.NewHealthCheckTimerWithInterfaces(clk, time.Nanosecond)
	records := []*kgo.Record{{Timestamp: clk.Now()}}
	c := s.newConsumeDelayTestConsumerWithHealthCheck(clk, healthCheck, records, 2*time.Nanosecond, time.Nanosecond)
	result := make(chan error, 1)

	go func() {
		result <- c.Run(s.T().Context())
	}()

	clk.BlockUntilTimers(1)
	clk.BlockUntilTickers(1)
	clk.Advance(2 * time.Nanosecond)

	s.Require().NoError(<-result)
}

func (s *PartitionManagerTestSuite) newConsumeDelayTestConsumer(clk clock.FakeClock, records []*kgo.Record, consumeDelay, healthTimeout time.Duration) consumer.Consumer {
	return s.newConsumeDelayTestConsumerWithHealthCheck(clk, clock.NewHealthCheckTimerWithInterfaces(clk, healthTimeout), records, consumeDelay, healthTimeout)
}

func (s *PartitionManagerTestSuite) newConsumeDelayTestConsumerWithHealthCheck(clk clock.FakeClock, healthCheck clock.HealthCheckTimer, records []*kgo.Record, consumeDelay, healthTimeout time.Duration) consumer.Consumer {
	reader := kafkaConsumerMocks.NewReader(s.T())
	handler := kafkaConsumerMocks.NewKafkaMessageHandler(s.T())
	metricWriter := metricMocks.NewWriter(s.T())

	reader.EXPECT().PollRecords(nil, 100).Return(fetchWithPartitionError(records, nil)).Once()
	reader.EXPECT().PollRecords(nil, 100).Return(clientClosedFetches()).Once()
	reader.EXPECT().AllowRebalance().Once()
	reader.EXPECT().CloseAllowingRebalance().Once()
	handler.EXPECT().Handle(records).Once()
	handler.EXPECT().Stop().Once()
	metricWriter.EXPECT().Write(mock.Anything, mock.Anything).Once()

	return consumer.NewConsumerWithInterfaces(
		log.NewLogger(),
		clk,
		healthCheck,
		handler,
		func(_ context.Context, _ *consumer.PartitionManager) (consumer.Reader, error) { return reader, nil },
		consumer.Settings{
			ConsumeDelay:   consumeDelay,
			Healthcheck:    health.HealthCheckSettings{Timeout: healthTimeout},
			IdleWaitTime:   time.Millisecond,
			MaxPollRecords: 100,
		},
		metricWriter,
		"test-topic",
		true,
		"test-consumer",
	)
}

func (s *PartitionManagerTestSuite) newCanceledConsumeDelayTestConsumer(clk clock.FakeClock, records []*kgo.Record, consumeDelay time.Duration) consumer.Consumer {
	reader := kafkaConsumerMocks.NewReader(s.T())
	handler := kafkaConsumerMocks.NewKafkaMessageHandler(s.T())
	metricWriter := metricMocks.NewWriter(s.T())

	reader.EXPECT().PollRecords(nil, 100).Return(fetchWithPartitionError(records, nil)).Once()
	reader.EXPECT().AllowRebalance().Once()
	reader.EXPECT().CloseAllowingRebalance().Once()
	handler.EXPECT().Stop().Once()
	metricWriter.EXPECT().Write(mock.Anything, mock.Anything).Once()

	return consumer.NewConsumerWithInterfaces(
		log.NewLogger(),
		clk,
		clock.NewHealthCheckTimerWithInterfaces(clk, time.Minute),
		handler,
		func(_ context.Context, _ *consumer.PartitionManager) (consumer.Reader, error) { return reader, nil },
		consumer.Settings{
			ConsumeDelay:   consumeDelay,
			Healthcheck:    health.HealthCheckSettings{Timeout: time.Minute},
			IdleWaitTime:   time.Millisecond,
			MaxPollRecords: 100,
		},
		metricWriter,
		"test-topic",
		true,
		"test-consumer",
	)
}
