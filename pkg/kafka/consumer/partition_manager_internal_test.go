package consumer

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/kafka"
	kafkaConsumerMocks "github.com/justtrackio/gosoline/pkg/kafka/consumer/mocks"
	"github.com/justtrackio/gosoline/pkg/log"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/metric"
	metricMocks "github.com/justtrackio/gosoline/pkg/metric/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kgo"
)

func TestPartitionManagerInternalTestSuite(t *testing.T) {
	suite.Run(t, new(PartitionManagerInternalTestSuite))
}

type PartitionManagerInternalTestSuite struct {
	suite.Suite
}

func (s *PartitionManagerInternalTestSuite) TestHandleReturnsWhenPartitionConsumerStops() {
	t := s.T()
	partitionConsumer := &PartitionConsumer{
		assignedBatch: make(chan []*kgo.Record),
		done:          make(chan struct{}),
	}
	manager := &PartitionManager{
		consumers: map[assignment]*PartitionConsumer{{"topic", 1}: partitionConsumer},
	}
	close(partitionConsumer.done)

	result := make(chan struct{})
	go func() {
		manager.Handle(t.Context(), "topic", 1, []*kgo.Record{{}})
		close(result)
	}()

	select {
	case <-result:
	case <-time.After(time.Second):
		t.Fatal("Handle blocked after the partition consumer stopped")
	}
}

func (s *PartitionManagerInternalTestSuite) TestIgnoresAssignmentsAfterStop() {
	t := s.T()
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))
	messageHandler := kafkaConsumerMocks.NewKafkaMessageHandler(t)
	metricWriter := metricMocks.NewWriter(t)

	messageHandler.EXPECT().Stop().Once()

	manager := newPartitionManager(logger, clock.NewFakeClock(), metricWriter, messageHandler, "test-consumer", 0, nil, 0)
	manager.Stop(context.Background())

	assert.NotPanics(t, func() {
		manager.OnPartitionsAssigned(context.Background(), nil, map[string][]int32{
			"topic": {1},
		})
	})
}

func (s *PartitionManagerInternalTestSuite) TestGroupedDelayDoesNotKeepHealthCheckHealthy() {
	t := s.T()
	clk := clock.NewFakeClock()
	healthCheck := clock.NewHealthCheckTimerWithInterfaces(clk, time.Minute)
	manager := &PartitionManager{
		clock:         clk,
		consumeDelay:  2 * time.Minute,
		healthCheck:   healthCheck,
		healthTimeout: time.Minute,
	}
	ctx, cancel := context.WithCancel(t.Context())
	result := make(chan bool, 1)

	go func() {
		result <- manager.delayConsume(ctx, make(chan struct{}), []*kgo.Record{{Timestamp: clk.Now()}})
	}()

	clk.BlockUntilTimers(1)
	clk.Advance(time.Minute + time.Millisecond)
	if healthCheck.IsHealthy() {
		t.Fatal("grouped consume delay kept the health check healthy")
	}

	cancel()
	select {
	case consumed := <-result:
		if consumed {
			t.Fatal("canceled consume delay unexpectedly completed")
		}
	case <-time.After(time.Second):
		t.Fatal("consume delay did not stop after cancellation")
	}
}

func (s *PartitionManagerInternalTestSuite) TestPartitionConsumerReportsWaitAndSleepDurationsSeparately() {
	t := s.T()
	clk := clock.NewFakeClock()
	client := kafkaConsumerMocks.NewPartitionClient(t)
	handler := kafkaConsumerMocks.NewKafkaMessageHandler(t)
	metricWriter := metricMocks.NewWriter(t)
	records := []*kgo.Record{{Offset: 1}}
	waitStart := clk.Now()
	clk.Advance(2 * time.Second)

	handler.EXPECT().Handle(records).Run(func([]*kgo.Record) {
		clk.Advance(4 * time.Second)
	}).Once()
	client.EXPECT().CommitRecords(mock.Anything, records[0]).Run(func(context.Context, ...*kgo.Record) {
		clk.Advance(5 * time.Second)
	}).Return(nil).Once()

	var expectedMetrics metric.Data
	expectedMetrics = append(expectedMetrics, kafka.MetricPair(kafka.DimensionConsumer, "consumer", metricNameWaitDuration, "topic", 1, 2000, metric.UnitMillisecondsAverage)...)
	expectedMetrics = append(expectedMetrics, kafka.MetricPair(kafka.DimensionConsumer, "consumer", metricNameSleepDuration, "topic", 1, 3000, metric.UnitMillisecondsAverage)...)
	expectedMetrics = append(expectedMetrics, kafka.MetricPair(kafka.DimensionConsumer, "consumer", metricNameProcessDuration, "topic", 1, 4000, metric.UnitMillisecondsAverage)...)
	expectedMetrics = append(expectedMetrics, kafka.MetricPair(kafka.DimensionConsumer, "consumer", metricNameCommitDuration, "topic", 1, 5000, metric.UnitMillisecondsAverage)...)
	metricWriter.EXPECT().Write(mock.Anything, expectedMetrics).Once()

	partitionConsumer := newPartitionConsumer(
		log.NewLogger(),
		clk,
		metricWriter,
		handler,
		client,
		"consumer",
		"topic",
		1,
		false,
		func(context.Context, <-chan struct{}, []*kgo.Record) bool {
			clk.Advance(3 * time.Second)

			return true
		},
	)

	keepConsuming, err := partitionConsumer.consumeBatch(t.Context(), records, waitStart)

	assert.NoError(t, err)
	assert.True(t, keepConsuming)
}

func (s *PartitionManagerInternalTestSuite) TestConsumerDefaultMetricsIncludeSleepDuration() {
	t := s.T()
	defaults := getConsumerDefaultMetrics("consumer", "topic")

	assert.Contains(t, defaults, &metric.Datum{
		Priority:   metric.PriorityHigh,
		MetricName: metricNameSleepDuration,
		Dimensions: metric.Dimensions{
			kafka.DimensionClientType: kafka.DimensionConsumer,
			kafka.DimensionClient:     "consumer",
			kafka.DimensionTopic:      "topic",
			kafka.DimensionPartition:  metric.DimensionDefault,
		},
		Unit: metric.UnitMillisecondsAverage,
		Kind: metric.KindDefault,
	})
}

func (s *PartitionManagerInternalTestSuite) TestHandleDelayedBatchDoesNotBlock() {
	t := s.T()
	client := kafkaConsumerMocks.NewPartitionClient(t)
	partition := map[string][]int32{"topic": {1}}
	partitionConsumer := &PartitionConsumer{
		assignedBatch: make(chan []*kgo.Record, 1),
		done:          make(chan struct{}),
		kafkaClient:   client,
		topic:         "topic",
		partition:     1,
		pauseFetch:    true,
	}
	manager := &PartitionManager{
		consumers: map[assignment]*PartitionConsumer{{"topic", 1}: partitionConsumer},
	}
	records := []*kgo.Record{{}}

	client.EXPECT().PauseFetchPartitions(partition).Return(nil).Once()
	client.EXPECT().ResumeFetchPartitions(partition).Once()

	result := make(chan struct{})
	go func() {
		manager.Handle(t.Context(), "topic", 1, records)
		close(result)
	}()

	select {
	case <-result:
	case <-time.After(time.Second):
		t.Fatal("Handle blocked while the partition consumer was delaying a batch")
	}

	partitionConsumer.resumeFetch()

	select {
	case assigned := <-partitionConsumer.assignedBatch:
		s.Equal(records, assigned)
	default:
		t.Fatal("delayed batch was not assigned")
	}
}

func (s *PartitionManagerInternalTestSuite) TestPartitionConsumerEmptyAssignmentDoesNotPauseOrQueue() {
	t := s.T()
	client := kafkaConsumerMocks.NewPartitionClient(t)
	partitionConsumer := &PartitionConsumer{
		assignedBatch: make(chan []*kgo.Record, 1),
		kafkaClient:   client,
		pauseFetch:    true,
	}

	partitionConsumer.Assign(t.Context(), nil)

	select {
	case assigned := <-partitionConsumer.assignedBatch:
		t.Fatalf("unexpected assigned records: %v", assigned)
	default:
	}
}

func (s *PartitionManagerInternalTestSuite) TestPartitionConsumerFinishedAssignmentResumesExistingPauseWithoutQueueing() {
	t := s.T()
	client := kafkaConsumerMocks.NewPartitionClient(t)
	partition := map[string][]int32{"topic": {1}}
	done := make(chan struct{})
	close(done)
	partitionConsumer := &PartitionConsumer{
		assignedBatch: make(chan []*kgo.Record, 1),
		done:          done,
		kafkaClient:   client,
		topic:         "topic",
		partition:     1,
		pauseFetch:    true,
	}
	partitionConsumer.paused.Store(true)

	client.EXPECT().ResumeFetchPartitions(partition).Once()

	partitionConsumer.Assign(t.Context(), []*kgo.Record{{}})

	select {
	case assigned := <-partitionConsumer.assignedBatch:
		t.Fatalf("unexpected assigned records: %v", assigned)
	default:
	}
}

func (s *PartitionManagerInternalTestSuite) TestPartitionConsumerClosesDoneBeforeFinalResume() {
	t := s.T()
	client := kafkaConsumerMocks.NewPartitionClient(t)
	partition := map[string][]int32{"topic": {1}}
	records := []*kgo.Record{{}}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	partitionConsumer := newPartitionConsumer(
		log.NewLogger(),
		clock.NewFakeClock(),
		nil,
		nil,
		client,
		"consumer",
		"topic",
		1,
		true,
		nil,
	)
	partitionConsumer.paused.Store(true)

	client.EXPECT().PauseFetchPartitions(partition).Return(nil).Maybe()
	client.EXPECT().ResumeFetchPartitions(partition).Run(func(map[string][]int32) {
		partitionConsumer.Assign(t.Context(), records)
	}).Once()

	assert.ErrorIs(t, partitionConsumer.Consume(ctx), context.Canceled)
	client.AssertNotCalled(t, "PauseFetchPartitions", partition)
	assert.False(t, partitionConsumer.paused.Load())

	select {
	case assigned := <-partitionConsumer.assignedBatch:
		t.Fatalf("unexpected assigned records after consume stopped: %v", assigned)
	default:
	}
}

func (s *PartitionManagerInternalTestSuite) TestPartitionConsumerCanceledAssignmentDoesNotPause() {
	t := s.T()
	client := kafkaConsumerMocks.NewPartitionClient(t)
	partitionConsumer := &PartitionConsumer{
		assignedBatch: make(chan []*kgo.Record),
		done:          make(chan struct{}),
		kafkaClient:   client,
		topic:         "topic",
		partition:     1,
		pauseFetch:    true,
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	partitionConsumer.Assign(ctx, []*kgo.Record{{}})
}

func (s *PartitionManagerInternalTestSuite) TestPartitionConsumerStoppedAssignmentDoesNotPause() {
	t := s.T()
	client := kafkaConsumerMocks.NewPartitionClient(t)
	partitionConsumer := &PartitionConsumer{
		assignedBatch: make(chan []*kgo.Record),
		stop:          make(chan struct{}),
		done:          make(chan struct{}),
		kafkaClient:   client,
		topic:         "topic",
		partition:     1,
		pauseFetch:    true,
	}
	close(partitionConsumer.stop)

	partitionConsumer.Assign(t.Context(), []*kgo.Record{{}})
}

func (s *PartitionManagerInternalTestSuite) TestPartitionConsumerResumesAfterDelayedBatchCommitted() {
	t := s.T()
	clk := clock.NewFakeClock()
	client := kafkaConsumerMocks.NewPartitionClient(t)
	handler := kafkaConsumerMocks.NewKafkaMessageHandler(t)
	metricWriter := metricMocks.NewWriter(t)
	partition := map[string][]int32{"topic": {1}}
	records := []*kgo.Record{{Offset: 1, Timestamp: clk.Now()}}
	handled := make(chan struct{})
	resumed := make(chan struct{})
	manager := &PartitionManager{
		clock:        clk,
		consumeDelay: time.Second,
	}

	client.EXPECT().PauseFetchPartitions(partition).Return(nil).Once()
	handler.EXPECT().Handle(records).Run(func([]*kgo.Record) {
		close(handled)
	}).Once()
	commitCall := client.EXPECT().CommitRecords(mock.Anything, records[0]).Return(nil).Once()
	client.EXPECT().ResumeFetchPartitions(partition).Run(func(map[string][]int32) {
		close(resumed)
	}).Once().NotBefore(commitCall)
	metricWriter.EXPECT().Write(mock.Anything, mock.Anything).Once()

	partitionConsumer := newPartitionConsumer(
		log.NewLogger(),
		clk,
		metricWriter,
		handler,
		client,
		"consumer",
		"topic",
		1,
		true,
		manager.delayConsume,
	)
	consumeResult := make(chan error, 1)
	go func() {
		consumeResult <- partitionConsumer.Consume(t.Context())
	}()

	partitionConsumer.Assign(t.Context(), records)
	clk.BlockUntilTimers(1)

	select {
	case <-handled:
		t.Fatal("batch handled before the consume delay elapsed")
	default:
	}
	select {
	case <-resumed:
		t.Fatal("partition resumed before the consume delay elapsed")
	default:
	}

	clk.Advance(time.Second)
	select {
	case <-resumed:
	case <-time.After(time.Second):
		t.Fatal("partition was not resumed after committing the delayed batch")
	}

	partitionConsumer.Stop()
	if err := waitForConsumeResult(t, consumeResult); err != nil {
		t.Fatalf("unexpected consume error: %v", err)
	}
}

func (s *PartitionManagerInternalTestSuite) TestPartitionConsumerRevocationDuringDelayResumesWithoutProcessing() {
	t := s.T()
	client := kafkaConsumerMocks.NewPartitionClient(t)
	handler := kafkaConsumerMocks.NewKafkaMessageHandler(t)
	metricWriter := metricMocks.NewWriter(t)
	partition := map[string][]int32{"topic": {1}}
	records := []*kgo.Record{{Offset: 1}}
	delayStarted := make(chan struct{})
	resumed := make(chan struct{})

	client.EXPECT().PauseFetchPartitions(partition).Return(nil).Once()
	client.EXPECT().ResumeFetchPartitions(partition).Run(func(map[string][]int32) {
		close(resumed)
	}).Once()

	partitionConsumer := newPartitionConsumer(
		log.NewLogger(),
		clock.NewFakeClock(),
		metricWriter,
		handler,
		client,
		"consumer",
		"topic",
		1,
		true,
		func(ctx context.Context, stopped <-chan struct{}, _ []*kgo.Record) bool {
			close(delayStarted)
			select {
			case <-ctx.Done():
			case <-stopped:
			}

			return false
		},
	)
	consumeResult := make(chan error, 1)
	go func() {
		consumeResult <- partitionConsumer.Consume(t.Context())
	}()

	partitionConsumer.Assign(t.Context(), records)
	<-delayStarted
	partitionConsumer.Stop()

	if err := waitForConsumeResult(t, consumeResult); err != nil {
		t.Fatalf("unexpected consume error: %v", err)
	}
	select {
	case <-resumed:
	default:
		t.Fatal("partition was not resumed after revocation canceled the delay")
	}
}

func (s *PartitionManagerInternalTestSuite) TestPartitionConsumerDrainProcessesQueuedBatchAndCommits() {
	t := s.T()
	client := kafkaConsumerMocks.NewPartitionClient(t)
	handler := kafkaConsumerMocks.NewKafkaMessageHandler(t)
	metricWriter := metricMocks.NewWriter(t)
	partition := map[string][]int32{"topic": {1}}
	records := []*kgo.Record{{Offset: 1}}

	client.EXPECT().PauseFetchPartitions(partition).Return(nil).Once()
	handler.EXPECT().Handle(records).Once()
	client.EXPECT().CommitRecords(mock.Anything, records[0]).Return(nil).Once()
	client.EXPECT().ResumeFetchPartitions(partition).Once()
	metricWriter.EXPECT().Write(mock.Anything, mock.Anything).Once()

	partitionConsumer := newPartitionConsumer(
		log.NewLogger(),
		clock.NewFakeClock(),
		metricWriter,
		handler,
		client,
		"consumer",
		"topic",
		1,
		true,
		func(context.Context, <-chan struct{}, []*kgo.Record) bool { return true },
	)

	partitionConsumer.Assign(t.Context(), records)
	partitionConsumer.Drain()

	s.NoError(partitionConsumer.Consume(t.Context()))
}

func (s *PartitionManagerInternalTestSuite) TestPartitionConsumerDrainHonorsContextCancellation() {
	t := s.T()
	client := kafkaConsumerMocks.NewPartitionClient(t)
	handler := kafkaConsumerMocks.NewKafkaMessageHandler(t)
	metricWriter := metricMocks.NewWriter(t)
	partition := map[string][]int32{"topic": {1}}
	records := []*kgo.Record{{Offset: 1}}
	delayStarted := make(chan struct{})
	ctx, cancel := context.WithCancel(t.Context())

	client.EXPECT().PauseFetchPartitions(partition).Return(nil).Once()
	client.EXPECT().ResumeFetchPartitions(partition).Once()

	partitionConsumer := newPartitionConsumer(
		log.NewLogger(),
		clock.NewFakeClock(),
		metricWriter,
		handler,
		client,
		"consumer",
		"topic",
		1,
		true,
		func(ctx context.Context, _ <-chan struct{}, _ []*kgo.Record) bool {
			close(delayStarted)
			<-ctx.Done()

			return false
		},
	)
	partitionConsumer.Assign(t.Context(), records)
	partitionConsumer.Drain()
	result := make(chan error, 1)
	go func() {
		result <- partitionConsumer.Consume(ctx)
	}()

	<-delayStarted
	cancel()

	s.NoError(waitForConsumeResult(t, result))
}

func (s *PartitionManagerInternalTestSuite) TestPartitionConsumerCommitFailureResumesPartition() {
	t := s.T()
	client := kafkaConsumerMocks.NewPartitionClient(t)
	handler := kafkaConsumerMocks.NewKafkaMessageHandler(t)
	metricWriter := metricMocks.NewWriter(t)
	partition := map[string][]int32{"topic": {1}}
	records := []*kgo.Record{{Offset: 1}}
	commitError := errors.New("commit failed")

	client.EXPECT().PauseFetchPartitions(partition).Return(nil).Once()
	handler.EXPECT().Handle(records).Once()
	client.EXPECT().CommitRecords(mock.Anything, records[0]).Return(commitError).Once()
	client.EXPECT().ResumeFetchPartitions(partition).Once()
	metricWriter.EXPECT().Write(mock.Anything, mock.Anything).Once()

	partitionConsumer := newPartitionConsumer(
		log.NewLogger(),
		clock.NewFakeClock(),
		metricWriter,
		handler,
		client,
		"consumer",
		"topic",
		1,
		true,
		func(context.Context, <-chan struct{}, []*kgo.Record) bool { return true },
	)
	consumeResult := make(chan error, 1)
	go func() {
		consumeResult <- partitionConsumer.Consume(t.Context())
	}()

	partitionConsumer.Assign(t.Context(), records)
	err := waitForConsumeResult(t, consumeResult)
	if !errors.Is(err, commitError) {
		t.Fatalf("expected commit error, got %v", err)
	}
}

func (s *PartitionManagerInternalTestSuite) TestPartitionConsumerStopsGracefullyOnRebalanceCommitFailure() {
	for name, commitError := range map[string]error{
		"illegal generation":       kerr.IllegalGeneration,
		"unknown member":           kerr.UnknownMemberID,
		"rebalance in progress":    kerr.RebalanceInProgress,
		"wrapped generation error": fmt.Errorf("commit rejected: %w", kerr.IllegalGeneration),
	} {
		s.Run(name, func() {
			t := s.T()
			client := kafkaConsumerMocks.NewPartitionClient(t)
			handler := kafkaConsumerMocks.NewKafkaMessageHandler(t)
			metricWriter := metricMocks.NewWriter(t)
			records := []*kgo.Record{{Offset: 1}}

			handler.EXPECT().Handle(records).Once()
			metricWriter.EXPECT().Write(mock.Anything, mock.Anything).Once()

			partitionConsumer := newPartitionConsumer(
				log.NewLogger(),
				clock.NewFakeClock(),
				metricWriter,
				handler,
				client,
				"consumer",
				"topic",
				1,
				false,
				func(context.Context, <-chan struct{}, []*kgo.Record) bool { return true },
			)
			client.EXPECT().CommitRecords(mock.Anything, records[0]).Run(func(context.Context, ...*kgo.Record) {
				partitionConsumer.Stop()
			}).Return(commitError).Once()
			manager := &PartitionManager{errors: make(chan error, 1)}
			result := make(chan error, 1)
			go func() {
				result <- manager.consumePartition(t.Context(), partitionConsumer)
			}()

			partitionConsumer.Assign(t.Context(), records)

			s.NoError(waitForConsumeResult(t, result))
			select {
			case err := <-manager.errors:
				t.Fatalf("unexpected reported rebalance commit error: %v", err)
			default:
			}
		})
	}
}

func (s *PartitionManagerInternalTestSuite) TestPartitionConsumerReportsRebalanceCommitFailureWhilePartitionIsAssigned() {
	t := s.T()
	client := kafkaConsumerMocks.NewPartitionClient(t)
	handler := kafkaConsumerMocks.NewKafkaMessageHandler(t)
	metricWriter := metricMocks.NewWriter(t)
	records := []*kgo.Record{{Offset: 1}}

	handler.EXPECT().Handle(records).Once()
	client.EXPECT().CommitRecords(mock.Anything, records[0]).Return(kerr.RebalanceInProgress).Once()
	metricWriter.EXPECT().Write(mock.Anything, mock.Anything).Once()

	partitionConsumer := newPartitionConsumer(
		log.NewLogger(),
		clock.NewFakeClock(),
		metricWriter,
		handler,
		client,
		"consumer",
		"topic",
		1,
		false,
		func(context.Context, <-chan struct{}, []*kgo.Record) bool { return true },
	)

	keepConsuming, err := partitionConsumer.consumeBatch(t.Context(), records, time.Now())

	assert.False(t, keepConsuming)
	assert.ErrorIs(t, err, kerr.RebalanceInProgress)
}

func (s *PartitionManagerInternalTestSuite) TestReportsPartitionConsumerFailure() {
	t := s.T()
	client := kafkaConsumerMocks.NewPartitionClient(t)
	handler := kafkaConsumerMocks.NewKafkaMessageHandler(t)
	metricWriter := metricMocks.NewWriter(t)
	records := []*kgo.Record{{Offset: 1}}
	commitError := errors.New("commit failed")

	handler.EXPECT().Handle(records).Once()
	client.EXPECT().CommitRecords(mock.Anything, records[0]).Return(commitError).Once()
	metricWriter.EXPECT().Write(mock.Anything, mock.Anything).Once()

	partitionConsumer := newPartitionConsumer(
		log.NewLogger(),
		clock.NewFakeClock(),
		metricWriter,
		handler,
		client,
		"consumer",
		"topic",
		1,
		false,
		func(context.Context, <-chan struct{}, []*kgo.Record) bool { return true },
	)
	manager := &PartitionManager{errors: make(chan error, 1)}
	result := make(chan error, 1)
	go func() {
		result <- manager.consumePartition(t.Context(), partitionConsumer)
	}()

	partitionConsumer.Assign(t.Context(), records)

	select {
	case err := <-manager.errors:
		if !errors.Is(err, commitError) {
			t.Fatalf("expected reported commit error, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("partition consumer failure was not reported")
	}

	if err := waitForConsumeResult(t, result); !errors.Is(err, commitError) {
		t.Fatalf("expected consume error, got %v", err)
	}
}

func (s *PartitionManagerInternalTestSuite) TestStopDoesNotLogReportedPartitionConsumerFailureAgain() {
	t := s.T()
	logger := logMocks.NewLogger(t)
	messageHandler := kafkaConsumerMocks.NewKafkaMessageHandler(t)
	metricWriter := metricMocks.NewWriter(t)
	client := kafkaConsumerMocks.NewPartitionClient(t)
	records := []*kgo.Record{{Offset: 1}}
	commitError := errors.New("commit failed")

	logger.EXPECT().Debug(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Once()
	messageHandler.EXPECT().Handle(records).Once()
	messageHandler.EXPECT().Stop().Once()
	client.EXPECT().CommitRecords(mock.Anything, records[0]).Return(commitError).Once()
	metricWriter.EXPECT().Write(mock.Anything, mock.Anything).Once()

	manager := newPartitionManager(logger, clock.NewFakeClock(), metricWriter, messageHandler, "consumer", 0, nil, 0)
	partitionConsumer := newPartitionConsumer(
		logger,
		clock.NewFakeClock(),
		metricWriter,
		messageHandler,
		client,
		"consumer",
		"topic",
		1,
		false,
		func(context.Context, <-chan struct{}, []*kgo.Record) bool { return true },
	)
	manager.consumers[assignment{"topic", 1}] = partitionConsumer
	manager.cfn.Go(func() error {
		return manager.consumePartition(t.Context(), partitionConsumer)
	})
	manager.Handle(t.Context(), "topic", 1, records)

	select {
	case err := <-manager.errors:
		if !errors.Is(err, commitError) {
			t.Fatalf("expected reported commit error, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("partition consumer failure was not reported")
	}

	manager.Stop(t.Context())
	logger.AssertNotCalled(t, "Error", mock.Anything, mock.Anything, mock.Anything)
}

func waitForConsumeResult(t *testing.T, result <-chan error) error {
	t.Helper()

	select {
	case err := <-result:
		return err
	case <-time.After(time.Second):
		t.Fatal("partition consumer did not stop")

		return nil
	}
}
