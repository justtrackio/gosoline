package consumer_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
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
	"github.com/justtrackio/gosoline/pkg/stream/health"
	"github.com/justtrackio/gosoline/pkg/test/matcher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kgo"
	"golang.org/x/sys/unix"
)

func TestConsumerTestSuite(t *testing.T) {
	suite.Run(t, new(ConsumerTestSuite))
}

type ConsumerTestSuite struct {
	suite.Suite

	logger           log.Logger
	clk              clock.Clock
	fakeClock        clock.FakeClock
	healthCheckTimer clock.HealthCheckTimer
	metricWriter     *metricMocks.Writer
}

func (s *ConsumerTestSuite) SetupTest() {
	s.fakeClock = clock.NewFakeClock()
	s.clk = s.fakeClock
	s.logger = log.NewLogger()
	s.healthCheckTimer = clock.NewHealthCheckTimerWithInterfaces(s.fakeClock, time.Minute)
	s.metricWriter = metricMocks.NewWriter(s.T())
}

func (s *ConsumerTestSuite) newTestConsumer(readerFactory consumer.ReaderFactory, settings *consumer.Settings) consumer.Consumer {
	return consumer.NewConsumerWithInterfaces(s.logger, s.clk, s.healthCheckTimer, readerFactory, settings, s.metricWriter, "test-topic", "test-consumer")
}

// newTestConsumerWithLogHandler builds a consumer logging into the given handler. Tests asserting on log
// output need this because the suite logger discards its messages and because a mocked logger is not usable
// either: the backoff executor derives its own logger via WithFields.
func (s *ConsumerTestSuite) newTestConsumerWithLogHandler(readerFactory consumer.ReaderFactory, settings *consumer.Settings, handler log.Handler) consumer.Consumer {
	logger := log.NewLoggerWithInterfaces(clock.NewRealClock(), []log.Handler{handler})

	return consumer.NewConsumerWithInterfaces(logger, s.clk, s.healthCheckTimer, readerFactory, settings, s.metricWriter, "test-topic", "test-consumer")
}

func (s *ConsumerTestSuite) TestConsumerRunStopReturns() {
	t := s.T()
	reader := kafkaConsumerMocks.NewReader(t)
	polling := make(chan struct{})
	stopped := make(chan struct{})

	reader.EXPECT().PollRecords(nil, 100).RunAndReturn(func(context.Context, int) kgo.Fetches {
		close(polling)
		<-stopped

		return clientClosedFetches()
	}).Once()
	reader.EXPECT().CloseAllowingRebalance().Once()

	readerFactory := func(_ context.Context, _ *consumer.PartitionManager) (consumer.Reader, error) {
		return reader, nil
	}

	c := s.newTestConsumer(readerFactory, &consumer.Settings{MaxPollRecords: 100, IdleWaitTime: 500 * time.Millisecond})

	errCh := make(chan error, 1)
	go func() {
		errCh <- c.Run(t.Context(), func(context.Context, *kgo.Record) bool { return true })
	}()

	<-polling
	c.Stop(t.Context())
	close(stopped)

	assert.NoError(t, <-errCh)
}

func (s *ConsumerTestSuite) TestConsumerRunIsUnhealthyWhenPollingIsStuck() {
	t := s.T()
	reader := kafkaConsumerMocks.NewReader(t)
	fakeClock := clock.NewFakeClock()
	polling := make(chan struct{})
	stopped := make(chan struct{})

	reader.EXPECT().PollRecords(nil, 100).RunAndReturn(func(context.Context, int) kgo.Fetches {
		close(polling)
		<-stopped

		return clientClosedFetches()
	}).Once()
	reader.EXPECT().CloseAllowingRebalance().Once()

	readerFactory := func(_ context.Context, _ *consumer.PartitionManager) (consumer.Reader, error) {
		return reader, nil
	}

	s.clk = fakeClock
	s.healthCheckTimer = clock.NewHealthCheckTimerWithInterfaces(fakeClock, time.Minute)
	c := s.newTestConsumer(readerFactory, &consumer.Settings{MaxPollRecords: 100})

	errCh := make(chan error, 1)
	go func() {
		errCh <- c.Run(t.Context(), func(context.Context, *kgo.Record) bool { return true })
	}()

	<-polling
	fakeClock.Advance(time.Minute + time.Nanosecond)
	assert.False(t, c.IsHealthy())

	c.Stop(t.Context())
	close(stopped)
	assert.NoError(t, <-errCh)
}

func (s *ConsumerTestSuite) TestConsumerStopBeforeRunReturnsWithoutCreatingReader() {
	t := s.T()
	readerFactoryCalled := false
	readerFactory := func(_ context.Context, _ *consumer.PartitionManager) (consumer.Reader, error) {
		readerFactoryCalled = true

		return nil, nil
	}
	c := s.newTestConsumer(readerFactory, &consumer.Settings{})

	c.Stop(t.Context())
	err := c.Run(t.Context(), func(context.Context, *kgo.Record) bool { return true })

	assert.NoError(t, err)
	assert.False(t, readerFactoryCalled)
}

func (s *ConsumerTestSuite) TestConsumerRunFailsWhenCalledTwice() {
	t := s.T()
	reader := kafkaConsumerMocks.NewReader(t)
	polling := make(chan struct{})
	stopped := make(chan struct{})

	reader.EXPECT().PollRecords(nil, 100).RunAndReturn(func(context.Context, int) kgo.Fetches {
		close(polling)
		<-stopped

		return clientClosedFetches()
	}).Once()
	reader.EXPECT().CloseAllowingRebalance().Once()
	readerFactory := func(_ context.Context, _ *consumer.PartitionManager) (consumer.Reader, error) {
		return reader, nil
	}
	c := s.newTestConsumer(readerFactory, &consumer.Settings{MaxPollRecords: 100})
	errCh := make(chan error, 1)

	go func() {
		errCh <- c.Run(t.Context(), func(context.Context, *kgo.Record) bool { return true })
	}()

	<-polling
	assert.EqualError(t, c.Run(t.Context(), func(context.Context, *kgo.Record) bool { return true }), "can not run a kafka consumer a second time")
	c.Stop(t.Context())
	close(stopped)
	assert.NoError(t, <-errCh)
}

// TestConsumerStopDrainsInFlightProcessingAndCommitsProcessedRecord verifies that a record which was handed to the
// callback is never committed without having been processed: as long as the caller's processing deadline is alive,
// the callback keeps its context even though the consumer itself is already shutting down.
func (s *ConsumerTestSuite) TestConsumerStopDrainsInFlightProcessingAndCommitsProcessedRecord() {
	t := s.T()
	reader := kafkaConsumerMocks.NewReader(t)
	record := &kgo.Record{Topic: "test-topic", Partition: 0, Offset: 1}
	processingStarted := make(chan struct{})
	releaseProcessing := make(chan struct{})
	var readerCtx context.Context

	reader.EXPECT().PollRecords(nil, 100).Return(fetchWithPartitionError([]*kgo.Record{record}, nil)).Once()
	reader.EXPECT().CommitRecords(matcher.Context, record).Run(func(ctx context.Context, _ ...*kgo.Record) {
		require.NoError(t, ctx.Err())
		require.NotNil(t, readerCtx)
		require.NoError(t, readerCtx.Err())
	}).Return(nil).Once()
	reader.EXPECT().AllowRebalance().Once()
	reader.EXPECT().CloseAllowingRebalance().Run(func() {
		require.NotNil(t, readerCtx)
		require.NoError(t, readerCtx.Err())
	}).Once()
	s.metricWriter.EXPECT().Write(matcher.Context, mock.Anything).Times(3)
	readerFactory := func(ctx context.Context, _ *consumer.PartitionManager) (consumer.Reader, error) {
		readerCtx = ctx

		return reader, nil
	}
	c := s.newTestConsumer(readerFactory, &consumer.Settings{MaxPollRecords: 100, GraceTime: time.Second, RunnerCount: 1})

	// The processing deadline stays alive for the whole test, so the drain is only ended by the callback returning.
	drainCtx, expireProcessingDeadline := context.WithCancel(t.Context())
	defer expireProcessingDeadline()

	errCh := make(chan error, 1)
	go func() {
		errCh <- c.Run(exec.WithDrainContext(t.Context(), drainCtx), func(ctx context.Context, _ *kgo.Record) bool {
			close(processingStarted)
			<-releaseProcessing
			// assert instead of require: this runs outside the test goroutine, where FailNow is illegal.
			assert.NoError(t, ctx.Err())

			return true
		})
	}()

	<-processingStarted
	c.Stop(t.Context())

	select {
	case err := <-errCh:
		t.Fatalf("consumer stopped before in-flight processing completed: %v", err)
	case <-time.After(100 * time.Millisecond):
	}

	close(releaseProcessing)
	assert.NoError(t, <-errCh)
	require.ErrorIs(t, readerCtx.Err(), context.Canceled)
}

func (s *ConsumerTestSuite) TestConsumerStopIgnoresCanceledCommit() {
	t := s.T()
	reader := kafkaConsumerMocks.NewReader(t)
	record := &kgo.Record{Topic: "test-topic", Partition: 0, Offset: 1}
	commitStarted := make(chan struct{})

	reader.EXPECT().PollRecords(nil, 100).Return(fetchWithPartitionError([]*kgo.Record{record}, nil)).Once()
	reader.EXPECT().CommitRecords(matcher.Context, record).RunAndReturn(func(ctx context.Context, _ ...*kgo.Record) error {
		close(commitStarted)
		<-ctx.Done()

		return ctx.Err()
	}).Once()
	reader.EXPECT().CloseAllowingRebalance().Once()
	s.metricWriter.EXPECT().Write(matcher.Context, mock.Anything).Once()

	c := s.newTestConsumer(func(_ context.Context, _ *consumer.PartitionManager) (consumer.Reader, error) {
		return reader, nil
	}, &consumer.Settings{MaxPollRecords: 100, GraceTime: time.Millisecond})

	errCh := make(chan error, 1)
	go func() {
		errCh <- c.Run(t.Context(), func(context.Context, *kgo.Record) bool { return true })
	}()

	<-commitStarted
	c.Stop(t.Context())

	select {
	case err := <-errCh:
		assert.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("consumer did not stop after the commit grace time elapsed")
	}
}

func (s *ConsumerTestSuite) TestConsumerRunProcessesDuplicatePartitionsSequentially() {
	t := s.T()
	reader := kafkaConsumerMocks.NewReader(t)
	first := &kgo.Record{Topic: "test-topic", Partition: 0, Offset: 1}
	second := &kgo.Record{Topic: "test-topic", Partition: 0, Offset: 2}
	firstStarted := make(chan struct{})
	secondStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	var processedMu sync.Mutex
	var processed []*kgo.Record

	reader.EXPECT().PollRecords(nil, 100).Return(fetchWithDuplicatePartition(first, second)).Once()
	reader.EXPECT().PollRecords(nil, 100).Return(clientClosedFetches()).Once()
	reader.EXPECT().CommitRecords(matcher.Context, first, second).Run(func(ctx context.Context, records ...*kgo.Record) {
		require.NoError(t, ctx.Err())
		assert.Equal(t, int64(2), records[len(records)-1].Offset)
	}).Return(nil).Once()
	reader.EXPECT().AllowRebalance().Once()
	reader.EXPECT().CloseAllowingRebalance().Once()
	s.metricWriter.EXPECT().Write(matcher.Context, mock.Anything).Times(3)

	readerFactory := func(_ context.Context, _ *consumer.PartitionManager) (consumer.Reader, error) {
		return reader, nil
	}
	c := s.newTestConsumer(readerFactory, &consumer.Settings{MaxPollRecords: 100, RunnerCount: 2, ProcessingMode: consumer.ProcessingModeOrdered})

	errCh := make(chan error, 1)
	go func() {
		errCh <- c.Run(t.Context(), func(_ context.Context, record *kgo.Record) bool {
			switch record {
			case first:
				close(firstStarted)
				<-releaseFirst
			case second:
				close(secondStarted)
			}

			processedMu.Lock()
			processed = append(processed, record)
			processedMu.Unlock()

			return true
		})
	}()

	<-firstStarted
	secondStartedBeforeFirstFinished := false
	select {
	case <-secondStarted:
		secondStartedBeforeFirstFinished = true
	case <-time.After(100 * time.Millisecond):
	}
	close(releaseFirst)

	require.NoError(t, <-errCh)
	assert.False(t, secondStartedBeforeFirstFinished)
	processedMu.Lock()
	assert.Equal(t, []*kgo.Record{first, second}, processed)
	processedMu.Unlock()
}

func (s *ConsumerTestSuite) TestConsumerRunRecoversProcessingPanicAndCommits() {
	for _, processingMode := range []consumer.ProcessingMode{consumer.ProcessingModeOrdered, consumer.ProcessingModeUnordered} {
		s.Run(string(processingMode), func() {
			t := s.T()
			reader := kafkaConsumerMocks.NewReader(t)
			metricWriter := metricMocks.NewWriter(t)
			record := &kgo.Record{Topic: "test-topic", Partition: 0, Offset: 1}
			var failedMetric atomic.Int32

			reader.EXPECT().PollRecords(nil, 100).Return(fetchWithPartitionError([]*kgo.Record{record}, nil)).Once()
			reader.EXPECT().PollRecords(nil, 100).Return(clientClosedFetches()).Once()
			reader.EXPECT().CommitRecords(matcher.Context, record).Return(nil).Once()
			reader.EXPECT().AllowRebalance().Once()
			reader.EXPECT().CloseAllowingRebalance().Once()
			metricWriter.EXPECT().Write(matcher.Context, mock.Anything).Run(func(_ context.Context, data metric.Data) {
				for _, datum := range data {
					if datum.MetricName == "RecordsConsumedFailed" && datum.Value == 1 {
						failedMetric.Add(1)
					}
				}
			}).Times(3)

			c := consumer.NewConsumerWithInterfaces(
				s.logger,
				s.clk,
				s.healthCheckTimer,
				func(_ context.Context, _ *consumer.PartitionManager) (consumer.Reader, error) {
					return reader, nil
				},
				&consumer.Settings{MaxPollRecords: 100, ProcessingMode: processingMode},
				metricWriter,
				"test-topic",
				"test-consumer",
			)

			err := c.Run(t.Context(), func(context.Context, *kgo.Record) bool {
				panic("processing failed")
			})

			assert.NoError(t, err)
			assert.Equal(t, int32(1), failedMetric.Load())
		})
	}
}

func (s *ConsumerTestSuite) TestConsumerStopInterruptsIdleWait() {
	t := s.T()
	reader := kafkaConsumerMocks.NewReader(t)
	waiting := make(chan struct{})

	reader.EXPECT().PollRecords(nil, 100).Return(kgo.Fetches{}).Once()
	reader.EXPECT().AllowRebalance().Once()
	reader.EXPECT().CloseAllowingRebalance().Once()
	s.metricWriter.EXPECT().Write(matcher.Context, mock.Anything).Run(func(ctx context.Context, _ metric.Data) {
		close(waiting)
		<-ctx.Done()
	}).Once()
	readerFactory := func(_ context.Context, _ *consumer.PartitionManager) (consumer.Reader, error) {
		return reader, nil
	}
	s.clk = clock.NewRealClock()
	s.healthCheckTimer = clock.NewHealthCheckTimerWithInterfaces(s.clk, time.Minute)
	c := s.newTestConsumer(readerFactory, &consumer.Settings{MaxPollRecords: 100, IdleWaitTime: time.Hour})
	errCh := make(chan error, 1)
	go func() {
		errCh <- c.Run(t.Context(), func(context.Context, *kgo.Record) bool { return true })
	}()

	<-waiting
	c.Stop(t.Context())

	select {
	case err := <-errCh:
		assert.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("consumer did not interrupt idle wait during shutdown")
	}
}

func (s *ConsumerTestSuite) TestConsumerRunProcessesRecordsUnorderedConcurrentlyAndTracksFailures() {
	t := s.T()
	reader := kafkaConsumerMocks.NewReader(t)
	first := &kgo.Record{Topic: "test-topic", Partition: 0, Offset: 1}
	second := &kgo.Record{Topic: "test-topic", Partition: 1, Offset: 1}
	started := make(chan struct{}, 2)
	release := make(chan struct{})
	var failedMetric atomic.Int32

	reader.EXPECT().PollRecords(nil, 100).Return(fetchWithPartitions(first, second)).Once()
	reader.EXPECT().PollRecords(nil, 100).Return(clientClosedFetches()).Once()
	reader.EXPECT().CommitRecords(matcher.Context, mock.Anything, mock.Anything).Run(func(_ context.Context, records ...*kgo.Record) {
		assert.ElementsMatch(t, []*kgo.Record{first, second}, records)
	}).Return(nil).Once()
	reader.EXPECT().AllowRebalance().Once()
	reader.EXPECT().CloseAllowingRebalance().Once()
	s.metricWriter.EXPECT().Write(matcher.Context, mock.Anything).Run(func(_ context.Context, data metric.Data) {
		for _, datum := range data {
			if datum.MetricName == "RecordsConsumedFailed" && datum.Value == 1 {
				failedMetric.Add(1)
			}
		}
	}).Times(3)

	c := s.newTestConsumer(func(_ context.Context, _ *consumer.PartitionManager) (consumer.Reader, error) {
		return reader, nil
	}, &consumer.Settings{MaxPollRecords: 100, RunnerCount: 2, ProcessingMode: consumer.ProcessingModeUnordered})

	var processed atomic.Int32
	errCh := make(chan error, 1)
	go func() {
		errCh <- c.Run(t.Context(), func(_ context.Context, record *kgo.Record) bool {
			processed.Add(1)
			started <- struct{}{}
			<-release

			return record == first
		})
	}()

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("unordered processing did not start")
	}
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("unordered processing did not run concurrently")
	}
	close(release)

	assert.NoError(t, <-errCh)
	assert.Equal(t, int32(2), processed.Load())
	assert.Equal(t, int32(1), failedMetric.Load())
}

func (s *ConsumerTestSuite) TestConsumerRunDrainsUnorderedInFlightProcessingWithoutStartingQueuedRecords() {
	t := s.T()
	reader := kafkaConsumerMocks.NewReader(t)
	first := &kgo.Record{Topic: "test-topic", Partition: 0, Offset: 1}
	second := &kgo.Record{Topic: "test-topic", Partition: 0, Offset: 2}
	third := &kgo.Record{Topic: "test-topic", Partition: 0, Offset: 3}
	firstStarted := make(chan struct{})
	shutdownStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	var processed []*kgo.Record
	var processedMu sync.Mutex

	reader.EXPECT().PollRecords(nil, 100).Return(fetchWithPartitions(first, second, third)).Once()
	reader.EXPECT().CommitRecords(matcher.Context, mock.Anything, mock.Anything).Run(func(_ context.Context, records ...*kgo.Record) {
		assert.ElementsMatch(t, []*kgo.Record{first, second}, records)
	}).Return(nil).Once()
	reader.EXPECT().AllowRebalance().Once()
	reader.EXPECT().CloseAllowingRebalance().Once()
	s.metricWriter.EXPECT().Write(matcher.Context, mock.Anything).Times(3)

	c := s.newTestConsumer(func(_ context.Context, _ *consumer.PartitionManager) (consumer.Reader, error) {
		return reader, nil
	}, &consumer.Settings{MaxPollRecords: 100, RunnerCount: 2, GraceTime: time.Second, ProcessingMode: consumer.ProcessingModeUnordered})

	errCh := make(chan error, 1)
	go func() {
		errCh <- c.Run(t.Context(), func(_ context.Context, record *kgo.Record) bool {
			switch record {
			case first:
				close(firstStarted)
				<-releaseFirst
			case second:
				<-firstStarted
				c.Stop(t.Context())
				close(shutdownStarted)
			case third:
				t.Error("queued record started processing after shutdown")
			}

			processedMu.Lock()
			processed = append(processed, record)
			processedMu.Unlock()

			return true
		})
	}()

	<-shutdownStarted

	select {
	case err := <-errCh:
		t.Fatalf("consumer stopped before in-flight processing completed: %v", err)
	case <-time.After(100 * time.Millisecond):
	}

	close(releaseFirst)
	assert.NoError(t, <-errCh)

	processedMu.Lock()
	assert.ElementsMatch(t, []*kgo.Record{first, second}, processed)
	processedMu.Unlock()
}

// TestConsumerRunCommitsGapFreePrefixOnShutdownWithConcurrentRunners guards the invariant that makes
// committing safe: the committed records always form a gap free prefix per topic partition. Since
// CommitRecords commits the highest offset per partition, a hole in the committed set would mark the
// records below it as consumed even though they were never processed.
//
// Three records are in flight when the shutdown starts and they complete in reverse offset order, so a
// higher offset is recorded before the lower ones. The queued records above them must not be started.
func (s *ConsumerTestSuite) TestConsumerRunCommitsGapFreePrefixOnShutdownWithConcurrentRunners() {
	t := s.T()
	reader := kafkaConsumerMocks.NewReader(t)
	records := make([]*kgo.Record, 0, 5)
	for offset := int64(1); offset <= 5; offset++ {
		records = append(records, &kgo.Record{Topic: "test-topic", Partition: 0, Offset: offset})
	}

	firstStarted := make(chan struct{})
	secondStarted := make(chan struct{})
	shutdownStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	releaseSecond := make(chan struct{})

	var committed []*kgo.Record
	reader.EXPECT().PollRecords(nil, 100).Return(fetchWithPartitions(records...)).Once()
	reader.EXPECT().CommitRecords(matcher.Context, mock.Anything, mock.Anything, mock.Anything).Run(func(_ context.Context, records ...*kgo.Record) {
		committed = records
	}).Return(nil).Once()
	reader.EXPECT().AllowRebalance().Once()
	reader.EXPECT().CloseAllowingRebalance().Once()
	s.metricWriter.EXPECT().Write(matcher.Context, mock.Anything).Times(3)

	// runnerCount 3 keeps records 1 to 3 in flight while records 4 and 5 wait for a free runner.
	c := s.newTestConsumer(func(_ context.Context, _ *consumer.PartitionManager) (consumer.Reader, error) {
		return reader, nil
	}, &consumer.Settings{MaxPollRecords: 100, RunnerCount: 3, GraceTime: time.Second, ProcessingMode: consumer.ProcessingModeUnordered})

	errCh := make(chan error, 1)
	go func() {
		errCh <- c.Run(t.Context(), func(_ context.Context, record *kgo.Record) bool {
			switch record.Offset {
			case 1:
				close(firstStarted)
				<-releaseFirst
			case 2:
				close(secondStarted)
				<-releaseSecond
			case 3:
				// Only stop once the two lower offsets are provably in flight, so the shutdown really
				// races against them instead of preventing their admission.
				<-firstStarted
				<-secondStarted
				c.Stop(t.Context())
				close(shutdownStarted)
			default:
				assert.Failf(t, "queued record started processing after shutdown", "offset %d", record.Offset)
			}

			return true
		})
	}()

	<-shutdownStarted

	// Complete the in-flight records in reverse offset order: the highest offset is already recorded, so
	// dropping either of the lower ones would produce exactly the gap this test guards against.
	close(releaseSecond)
	close(releaseFirst)
	assert.NoError(t, <-errCh)

	require.NotEmpty(t, committed)
	assertGapFreePrefix(t, records, committed)
	assert.ElementsMatch(t, records[:3], committed)
}

// TestConsumerRunOrderedStopsAtRecordBoundary verifies that ordered processing stops between two records
// of a partition rather than in the middle of one, and that the already processed prefix is committed.
func (s *ConsumerTestSuite) TestConsumerRunOrderedStopsAtRecordBoundary() {
	t := s.T()
	reader := kafkaConsumerMocks.NewReader(t)
	firstPartition := []*kgo.Record{
		{Topic: "test-topic", Partition: 0, Offset: 1},
		{Topic: "test-topic", Partition: 0, Offset: 2},
		{Topic: "test-topic", Partition: 0, Offset: 3},
	}
	secondPartition := &kgo.Record{Topic: "test-topic", Partition: 1, Offset: 1}

	partitionStarted := make(chan struct{})
	shutdownStarted := make(chan struct{})
	releasePartition := make(chan struct{})

	var committed []*kgo.Record
	reader.EXPECT().PollRecords(nil, 100).Return(fetchWithPartitions(append(firstPartition, secondPartition)...)).Once()
	reader.EXPECT().CommitRecords(matcher.Context, mock.Anything, mock.Anything).Run(func(_ context.Context, records ...*kgo.Record) {
		committed = records
	}).Return(nil).Once()
	reader.EXPECT().AllowRebalance().Once()
	reader.EXPECT().CloseAllowingRebalance().Once()
	s.metricWriter.EXPECT().Write(matcher.Context, mock.Anything).Times(3)

	c := s.newTestConsumer(func(_ context.Context, _ *consumer.PartitionManager) (consumer.Reader, error) {
		return reader, nil
	}, &consumer.Settings{MaxPollRecords: 100, RunnerCount: 2, GraceTime: time.Second, ProcessingMode: consumer.ProcessingModeOrdered})

	errCh := make(chan error, 1)
	go func() {
		errCh <- c.Run(t.Context(), func(_ context.Context, record *kgo.Record) bool {
			switch {
			case record == secondPartition:
				<-partitionStarted
				c.Stop(t.Context())
				close(shutdownStarted)
			case record.Offset == 1:
				close(partitionStarted)
				<-releasePartition
			default:
				assert.Failf(t, "record was processed after shutdown", "offset %d", record.Offset)
			}

			return true
		})
	}()

	<-shutdownStarted
	close(releasePartition)
	assert.NoError(t, <-errCh)

	assert.ElementsMatch(t, []*kgo.Record{firstPartition[0], secondPartition}, committed)
}

// TestConsumerRunCancelsInFlightProcessingAtProcessingDeadline verifies that a callback which ignores the
// shutdown is not drained forever: it survives the shutdown itself, but once the processing deadline the caller
// attached expires, its context is cancelled.
func (s *ConsumerTestSuite) TestConsumerRunCancelsInFlightProcessingAtProcessingDeadline() {
	t := s.T()
	reader := kafkaConsumerMocks.NewReader(t)
	record := &kgo.Record{Topic: "test-topic", Partition: 0, Offset: 1}
	processingStarted := make(chan struct{})
	processingCancelled := make(chan struct{})

	reader.EXPECT().PollRecords(nil, 100).Return(fetchWithPartitionError([]*kgo.Record{record}, nil)).Once()
	reader.EXPECT().CommitRecords(matcher.Context, record).Return(nil).Once()
	reader.EXPECT().AllowRebalance().Once()
	reader.EXPECT().CloseAllowingRebalance().Once()
	s.metricWriter.EXPECT().Write(matcher.Context, mock.Anything).Times(3)

	c := s.newTestConsumer(func(_ context.Context, _ *consumer.PartitionManager) (consumer.Reader, error) {
		return reader, nil
	}, &consumer.Settings{MaxPollRecords: 100, RunnerCount: 1, GraceTime: time.Minute})

	drainCtx, expireProcessingDeadline := context.WithCancel(t.Context())
	defer expireProcessingDeadline()

	errCh := make(chan error, 1)
	go func() {
		errCh <- c.Run(exec.WithDrainContext(t.Context(), drainCtx), func(ctx context.Context, _ *kgo.Record) bool {
			close(processingStarted)
			// A callback that never returns on its own must still be released by the processing deadline.
			<-ctx.Done()
			close(processingCancelled)

			return true
		})
	}()

	<-processingStarted
	c.Stop(t.Context())

	assert.Never(t, func() bool {
		select {
		case <-processingCancelled:
			return true
		default:
			return false
		}
	}, 50*time.Millisecond, 5*time.Millisecond,
		"in-flight processing must survive the shutdown until the processing deadline expires")

	expireProcessingDeadline()

	assert.NoError(t, <-errCh)
}

// TestConsumerRunPropagatesCancellationWithoutProcessingDeadline verifies that a caller which attaches no drain
// context keeps full control over its shutdown: cancellation reaches the callback right away instead of being
// delayed by a grace period the consumer invented on its behalf.
func (s *ConsumerTestSuite) TestConsumerRunPropagatesCancellationWithoutProcessingDeadline() {
	t := s.T()
	reader := kafkaConsumerMocks.NewReader(t)
	record := &kgo.Record{Topic: "test-topic", Partition: 0, Offset: 1}
	processingStarted := make(chan struct{})

	reader.EXPECT().PollRecords(nil, 100).Return(fetchWithPartitionError([]*kgo.Record{record}, nil)).Once()
	reader.EXPECT().CommitRecords(matcher.Context, record).Return(nil).Once()
	reader.EXPECT().AllowRebalance().Once()
	reader.EXPECT().CloseAllowingRebalance().Once()
	s.metricWriter.EXPECT().Write(matcher.Context, mock.Anything).Times(3)

	c := s.newTestConsumer(func(_ context.Context, _ *consumer.PartitionManager) (consumer.Reader, error) {
		return reader, nil
	}, &consumer.Settings{MaxPollRecords: 100, RunnerCount: 1, GraceTime: time.Minute})

	errCh := make(chan error, 1)
	go func() {
		errCh <- c.Run(t.Context(), func(ctx context.Context, _ *kgo.Record) bool {
			close(processingStarted)
			<-ctx.Done()

			return true
		})
	}()

	<-processingStarted
	// Nothing but the shutdown itself happens here: no clock is advanced and no deadline expires, so the callback
	// can only be released by the cancellation being propagated straight through.
	c.Stop(t.Context())

	assert.NoError(t, <-errCh)
}

// TestConsumerRunLogsRecordsAffectedByExpiredProcessingDeadline guards the transparency of the shutdown case that
// loses work outright: records whose processing was cut short by the expired processing deadline are committed
// regardless and therefore never consumed again. Only the log tells how much work that affected.
func (s *ConsumerTestSuite) TestConsumerRunLogsRecordsAffectedByExpiredProcessingDeadline() {
	t := s.T()
	reader := kafkaConsumerMocks.NewReader(t)
	records := []*kgo.Record{
		{Topic: "test-topic", Partition: 0, Offset: 1},
		{Topic: "test-topic", Partition: 0, Offset: 2},
		{Topic: "test-topic", Partition: 0, Offset: 3},
	}
	processingStarted := make(chan struct{})
	handler := &recordingLogHandler{}

	var committed []*kgo.Record

	reader.EXPECT().PollRecords(nil, 100).Return(fetchWithPartitionError(records, nil)).Once()
	reader.EXPECT().CommitRecords(matcher.Context, mock.Anything).RunAndReturn(func(_ context.Context, committedRecords ...*kgo.Record) error {
		committed = committedRecords

		return nil
	}).Once()
	reader.EXPECT().AllowRebalance().Once()
	reader.EXPECT().CloseAllowingRebalance().Once()
	s.metricWriter.EXPECT().Write(matcher.Context, mock.Anything).Times(3)

	// A single runner keeps the two remaining records queued behind the blocking one, so they are never
	// admitted and end up in the redelivered part of the log message.
	c := s.newTestConsumerWithLogHandler(
		func(_ context.Context, _ *consumer.PartitionManager) (consumer.Reader, error) {
			return reader, nil
		},
		&consumer.Settings{MaxPollRecords: 100, RunnerCount: 1, GraceTime: time.Minute, ProcessingMode: consumer.ProcessingModeUnordered},
		handler,
	)

	drainCtx, expireProcessingDeadline := context.WithCancel(t.Context())
	defer expireProcessingDeadline()

	errCh := make(chan error, 1)
	go func() {
		errCh <- c.Run(exec.WithDrainContext(t.Context(), drainCtx), func(ctx context.Context, _ *kgo.Record) bool {
			close(processingStarted)
			<-ctx.Done()

			return true
		})
	}()

	<-processingStarted
	c.Stop(t.Context())
	expireProcessingDeadline()

	assert.NoError(t, <-errCh)

	assert.True(t, handler.hasErrorContaining("processing deadline expired, cancelling in-flight record processing"),
		"expected the cancellation itself to be logged, got: %v", handler.messages())
	assert.True(t, handler.hasErrorContaining("1 of 3 records were handed to the callback (0 of them failed) and are committed regardless, the remaining 2 were skipped and are consumed again"),
		"expected the affected record counts to be logged, got: %v", handler.messages())
	assert.Equal(t, []*kgo.Record{records[0]}, committed,
		"the record cut short by the processing deadline is committed regardless, the queued ones are not")
}

// assertGapFreePrefix asserts that the committed records contain every polled record of a partition whose
// offset is below the highest committed offset of that partition. CommitRecords commits the highest offset
// per partition, so any missing record in between would silently be marked as consumed.
func assertGapFreePrefix(t *testing.T, polled []*kgo.Record, committed []*kgo.Record) {
	t.Helper()

	highest := make(map[int32]int64)
	for _, record := range committed {
		if offset, ok := highest[record.Partition]; !ok || record.Offset > offset {
			highest[record.Partition] = record.Offset
		}
	}

	for _, record := range polled {
		if record.Offset > highest[record.Partition] {
			continue
		}

		assert.Containsf(t, committed, record, "committing partition %d up to offset %d skips offset %d", record.Partition, highest[record.Partition], record.Offset)
	}
}

func (s *ConsumerTestSuite) TestConsumerRunProcessesOrderedPartitionsConcurrentlyAndTracksFailures() {
	t := s.T()
	reader := kafkaConsumerMocks.NewReader(t)
	first := &kgo.Record{Topic: "test-topic", Partition: 0, Offset: 1}
	second := &kgo.Record{Topic: "test-topic", Partition: 1, Offset: 1}
	started := make(chan struct{}, 2)
	release := make(chan struct{})
	var failedMetric atomic.Int32

	reader.EXPECT().PollRecords(nil, 100).Return(fetchWithPartitions(first, second)).Once()
	reader.EXPECT().PollRecords(nil, 100).Return(clientClosedFetches()).Once()
	reader.EXPECT().CommitRecords(matcher.Context, mock.Anything, mock.Anything).Run(func(_ context.Context, records ...*kgo.Record) {
		assert.ElementsMatch(t, []*kgo.Record{first, second}, records)
	}).Return(nil).Once()
	reader.EXPECT().AllowRebalance().Once()
	reader.EXPECT().CloseAllowingRebalance().Once()
	s.metricWriter.EXPECT().Write(matcher.Context, mock.Anything).Run(func(_ context.Context, data metric.Data) {
		for _, datum := range data {
			if datum.MetricName == "RecordsConsumedFailed" && datum.Value == 1 {
				failedMetric.Add(1)
			}
		}
	}).Times(3)

	c := s.newTestConsumer(func(_ context.Context, _ *consumer.PartitionManager) (consumer.Reader, error) {
		return reader, nil
	}, &consumer.Settings{MaxPollRecords: 100, RunnerCount: 2, ProcessingMode: consumer.ProcessingModeOrdered})

	errCh := make(chan error, 1)
	go func() {
		errCh <- c.Run(t.Context(), func(_ context.Context, record *kgo.Record) bool {
			started <- struct{}{}
			<-release

			return record == first
		})
	}()

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("ordered processing did not start")
	}
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("ordered partitions did not run concurrently")
	}
	close(release)

	assert.NoError(t, <-errCh)
	assert.Equal(t, int32(1), failedMetric.Load())
}

func (s *ConsumerTestSuite) TestConsumerRunReturnsCommitError() {
	t := s.T()
	reader := kafkaConsumerMocks.NewReader(t)
	commitErr := errors.New("commit failed")
	record := &kgo.Record{Topic: "test-topic", Partition: 0, Offset: 1}

	reader.EXPECT().PollRecords(nil, 100).Return(fetchWithPartitionError([]*kgo.Record{record}, nil)).Once()
	reader.EXPECT().CommitRecords(matcher.Context, record).Return(commitErr).Once()
	reader.EXPECT().CloseAllowingRebalance().Once()
	s.metricWriter.EXPECT().Write(matcher.Context, mock.Anything).Once()

	c := s.newTestConsumer(func(_ context.Context, _ *consumer.PartitionManager) (consumer.Reader, error) {
		return reader, nil
	}, &consumer.Settings{MaxPollRecords: 100, Backoff: exec.BackoffSettings{MaxAttempts: 1}})

	err := c.Run(t.Context(), func(context.Context, *kgo.Record) bool { return true })
	require.Error(t, err)
	assert.ErrorIs(t, err, commitErr)
	assert.Contains(t, err.Error(), "failed to commit records")
}

func (s *ConsumerTestSuite) TestConsumerRunReturnsReaderFactoryError() {
	t := s.T()
	metricWriter := metricMocks.NewWriter(t)
	factoryErr := errors.New("reader factory failed")
	var readerCtx context.Context

	c := consumer.NewConsumerWithInterfaces(
		log.NewLogger(),
		clock.NewFakeClock(),
		clock.NewHealthCheckTimerWithInterfaces(clock.NewFakeClock(), time.Minute),
		func(ctx context.Context, _ *consumer.PartitionManager) (consumer.Reader, error) {
			readerCtx = ctx

			return nil, factoryErr
		},
		&consumer.Settings{},
		metricWriter,
		"test-topic",
		"test-consumer",
	)

	err := c.Run(t.Context(), func(context.Context, *kgo.Record) bool { return true })
	require.Error(t, err)
	assert.ErrorIs(t, err, factoryErr)
	assert.ErrorIs(t, readerCtx.Err(), context.Canceled)
}

func (s *ConsumerTestSuite) TestConsumerRunReturnsParentCancellationWithoutCreatingReader() {
	t := s.T()
	metricWriter := metricMocks.NewWriter(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	called := false

	c := consumer.NewConsumerWithInterfaces(
		log.NewLogger(),
		clock.NewFakeClock(),
		clock.NewHealthCheckTimerWithInterfaces(clock.NewFakeClock(), time.Minute),
		func(context.Context, *consumer.PartitionManager) (consumer.Reader, error) {
			called = true

			return nil, nil
		},
		&consumer.Settings{},
		metricWriter,
		"test-topic",
		"test-consumer",
	)

	err := c.Run(ctx, func(context.Context, *kgo.Record) bool { return true })
	assert.ErrorIs(t, err, context.Canceled)
	assert.False(t, called)
}

func (s *ConsumerTestSuite) TestConsumerStopCanBeCalledMoreThanOnce() {
	t := s.T()
	metricWriter := metricMocks.NewWriter(t)
	called := false

	c := consumer.NewConsumerWithInterfaces(
		log.NewLogger(),
		clock.NewFakeClock(),
		clock.NewHealthCheckTimerWithInterfaces(clock.NewFakeClock(), time.Minute),
		func(context.Context, *consumer.PartitionManager) (consumer.Reader, error) {
			called = true

			return nil, nil
		},
		&consumer.Settings{},
		metricWriter,
		"test-topic",
		"test-consumer",
	)

	c.Stop(t.Context())
	c.Stop(t.Context())

	assert.NoError(t, c.Run(t.Context(), func(context.Context, *kgo.Record) bool { return true }))
	assert.False(t, called)
}

func (s *ConsumerTestSuite) TestConsumerIsHealthy() {
	t := s.T()
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
		readerFactory,
		&consumer.Settings{
			MaxPollRecords: 100,
			IdleWaitTime:   500 * time.Millisecond,
		},
		metricWriter,
		"test-topic",
		"test-consumer",
	)

	// Initially healthy (timer hasn't expired)
	assert.True(t, c.IsHealthy())

	// Advance past the timeout
	fakeClock.Advance(2 * time.Minute)

	// Should be unhealthy after timeout
	assert.False(t, c.IsHealthy())
}

func (s *ConsumerTestSuite) TestCheckKafkaUnknownTopicError() {
	t := s.T()
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
func (s *ConsumerTestSuite) TestConsumerRunFailsFastOnUnknownTopic() {
	t := s.T()
	for name, partitionErr := range map[string]error{
		"UnknownTopicOrPartition": kerr.UnknownTopicOrPartition,
		"UnknownTopicID":          kerr.UnknownTopicID,
	} {
		t.Run(name, func(t *testing.T) {
			reader := kafkaConsumerMocks.NewReader(t)
			metricWriter := metricMocks.NewWriter(t)

			reader.EXPECT().PollRecords(nil, 100).Return(fetchWithPartitionError(nil, partitionErr)).Once()
			reader.EXPECT().CloseAllowingRebalance().Once()

			readerFactory := func(_ context.Context, _ *consumer.PartitionManager) (consumer.Reader, error) {
				return reader, nil
			}

			fakeClock := clock.NewFakeClock()
			healthCheckTimer := clock.NewHealthCheckTimerWithInterfaces(fakeClock, time.Minute)

			c := consumer.NewConsumerWithInterfaces(
				log.NewLogger(),
				fakeClock,
				healthCheckTimer,
				readerFactory,
				&consumer.Settings{
					MaxPollRecords: 100,
					IdleWaitTime:   500 * time.Millisecond,
				},
				metricWriter,
				"missing-topic",
				"test-consumer",
			)

			err := c.Run(t.Context(), func(context.Context, *kgo.Record) bool { return true })
			require.Error(t, err)
			assert.True(t, errors.Is(err, partitionErr), "expected error to wrap %v, got: %v", partitionErr, err)
		})
	}
}

// A retryable, non-unknown-topic fetch error (e.g. NOT_LEADER_FOR_PARTITION) must be ignored so that the
// records delivered alongside it are still processed and the consumer keeps running.
func (s *ConsumerTestSuite) TestConsumerRunIgnoresRetryableFetchError() {
	t := s.T()
	reader := kafkaConsumerMocks.NewReader(t)
	metricWriter := metricMocks.NewWriter(t)

	records := []*kgo.Record{{Value: []byte("payload")}}
	// The first poll returns a record alongside a retryable per-partition error; the second poll reports the
	// client as closed so Run leaves its loop.
	reader.EXPECT().PollRecords(nil, 100).Return(fetchWithPartitionError(records, kerr.NotLeaderForPartition)).Once()
	reader.EXPECT().PollRecords(nil, 100).Return(clientClosedFetches()).Once()
	reader.EXPECT().CommitRecords(matcher.Context, records[0]).Return(nil).Once()
	reader.EXPECT().AllowRebalance().Once()
	reader.EXPECT().CloseAllowingRebalance().Once()

	dims := metric.Dimensions{
		kafka.DimensionClientType: kafka.DimensionConsumer,
		kafka.DimensionClient:     "test-consumer",
		kafka.DimensionTopic:      "test-topic",
	}
	expectedMetrics := metric.Data{
		metric.NewMetricDatum("PollCount", dims, 1.0, metric.UnitCount, metric.PriorityHigh),
		metric.NewMetricDatum("PollDuration", dims, 0.0, metric.UnitMillisecondsAverage, metric.PriorityHigh),
		metric.NewMetricDatum("ProcessDuration", dims, 0.0, metric.UnitMillisecondsAverage, metric.PriorityHigh),
		metric.NewMetricDatum("RecordsConsumed", dims, 1.0, metric.UnitCount, metric.PriorityHigh),
	}
	metricWriter.EXPECT().Write(matcher.Context, expectedMetrics).Once()
	metricWriter.EXPECT().Write(matcher.Context, mock.Anything).Times(2)

	readerFactory := func(_ context.Context, _ *consumer.PartitionManager) (consumer.Reader, error) {
		return reader, nil
	}

	fakeClock := clock.NewFakeClock()
	healthCheckTimer := clock.NewHealthCheckTimerWithInterfaces(fakeClock, time.Minute)

	c := consumer.NewConsumerWithInterfaces(
		log.NewLogger(),
		fakeClock,
		healthCheckTimer,
		readerFactory,
		&consumer.Settings{
			MaxPollRecords: 100,
			IdleWaitTime:   500 * time.Millisecond,
		},
		metricWriter,
		"test-topic",
		"test-consumer",
	)

	var processed []*kgo.Record
	err := c.Run(t.Context(), func(_ context.Context, record *kgo.Record) bool {
		processed = append(processed, record)

		return true
	})
	assert.NoError(t, err, "a retryable non-unknown-topic fetch error should be ignored")
	assert.Equal(t, records, processed)
}

func (s *ConsumerTestSuite) TestConsumerRunProcessesRecordsWithRetryableTransportFetchError() {
	t := s.T()
	reader := kafkaConsumerMocks.NewReader(t)
	metricWriter := metricMocks.NewWriter(t)
	records := []*kgo.Record{
		{Topic: "test-topic", Partition: 0, Offset: 1, Value: []byte("first")},
		{Topic: "test-topic", Partition: 0, Offset: 2, Value: []byte("second")},
	}

	reader.EXPECT().PollRecords(nil, 100).Return(fetchWithPartitionError(records, unix.ECONNREFUSED)).Once()
	reader.EXPECT().PollRecords(nil, 100).Return(clientClosedFetches()).Once()
	reader.EXPECT().CommitRecords(matcher.Context, records[0], records[1]).Return(nil).Once()
	reader.EXPECT().AllowRebalance().Once()
	reader.EXPECT().CloseAllowingRebalance().Once()
	metricWriter.EXPECT().Write(matcher.Context, mock.Anything).Times(3)

	readerFactory := func(_ context.Context, _ *consumer.PartitionManager) (consumer.Reader, error) {
		return reader, nil
	}

	fakeClock := clock.NewFakeClock()
	healthCheckTimer := clock.NewHealthCheckTimerWithInterfaces(fakeClock, time.Minute)
	c := consumer.NewConsumerWithInterfaces(
		log.NewLogger(),
		fakeClock,
		healthCheckTimer,
		readerFactory,
		&consumer.Settings{
			MaxPollRecords: 100,
			IdleWaitTime:   500 * time.Millisecond,
		},
		metricWriter,
		"test-topic",
		"test-consumer",
	)

	var processed []*kgo.Record
	err := c.Run(t.Context(), func(_ context.Context, record *kgo.Record) bool {
		processed = append(processed, record)

		return true
	})

	assert.NoError(t, err, "a retryable transport fetch error should not discard records")
	assert.Equal(t, records, processed)
}

// A non-retryable fetch error must be surfaced so the executor treats it as permanent and fails the consumer.
func (s *ConsumerTestSuite) TestConsumerRunSurfacesNonRetryableFetchError() {
	t := s.T()
	reader := kafkaConsumerMocks.NewReader(t)
	metricWriter := metricMocks.NewWriter(t)

	// InvalidTopicException (code 17) is not retryable.
	reader.EXPECT().PollRecords(nil, 100).Return(fetchWithPartitionError(nil, kerr.InvalidTopicException)).Once()
	reader.EXPECT().CloseAllowingRebalance().Once()

	readerFactory := func(_ context.Context, _ *consumer.PartitionManager) (consumer.Reader, error) {
		return reader, nil
	}

	fakeClock := clock.NewFakeClock()
	healthCheckTimer := clock.NewHealthCheckTimerWithInterfaces(fakeClock, time.Minute)

	c := consumer.NewConsumerWithInterfaces(
		log.NewLogger(),
		fakeClock,
		healthCheckTimer,
		readerFactory,
		&consumer.Settings{
			MaxPollRecords: 100,
			IdleWaitTime:   500 * time.Millisecond,
		},
		metricWriter,
		"test-topic",
		"test-consumer",
	)

	err := c.Run(t.Context(), func(context.Context, *kgo.Record) bool { return true })
	require.Error(t, err)
	assert.True(t, errors.Is(err, kerr.InvalidTopicException), "expected error to wrap INVALID_TOPIC_EXCEPTION, got: %v", err)
}

// TestConsumerRunReturnsNoErrorWhenParentIsCanceledDuringIdleWait pins that a cancellation of the context Run
// was called with is a regular shutdown rather than a failure. The consumer runs as an input of a stream
// consumer, whose coffin cancels that context whenever any of its routines dies. Reporting the resulting
// context error would attach an unrelated "context canceled" to a shutdown that was caused by something else
// entirely.
func (s *ConsumerTestSuite) TestConsumerRunReturnsNoErrorWhenParentIsCanceledDuringIdleWait() {
	t := s.T()
	reader := kafkaConsumerMocks.NewReader(t)
	ctx, cancel := context.WithCancel(t.Context())
	waiting := make(chan struct{})

	// An empty fetch sends the loop into waitForRecords, where the hour long idle wait makes the cancellation
	// the only branch that can fire.
	reader.EXPECT().PollRecords(nil, 100).Return(kgo.Fetches{}).Once()
	reader.EXPECT().AllowRebalance().Once()
	reader.EXPECT().CloseAllowingRebalance().Once()
	s.metricWriter.EXPECT().Write(matcher.Context, mock.Anything).Run(func(ctx context.Context, _ metric.Data) {
		close(waiting)
		<-ctx.Done()
	}).Once()

	readerFactory := func(_ context.Context, _ *consumer.PartitionManager) (consumer.Reader, error) {
		return reader, nil
	}
	s.clk = clock.NewRealClock()
	s.healthCheckTimer = clock.NewHealthCheckTimerWithInterfaces(s.clk, time.Minute)
	c := s.newTestConsumer(readerFactory, &consumer.Settings{MaxPollRecords: 100, IdleWaitTime: time.Hour})

	errCh := make(chan error, 1)
	go func() {
		errCh <- c.Run(ctx, func(context.Context, *kgo.Record) bool { return true })
	}()

	<-waiting
	cancel()

	select {
	case err := <-errCh:
		assert.NoError(t, err, "a canceled parent context must not be reported as a consumer error")
	case <-time.After(time.Second):
		t.Fatal("consumer did not interrupt idle wait when the parent context was canceled")
	}
}

// TestConsumerRunReturnsNoErrorWhenParentIsCanceledBetweenPolls covers the same cancellation as
// TestConsumerRunReturnsNoErrorWhenParentIsCanceledDuringIdleWait, but observed at the top of the poll loop
// instead of during the idle wait. Which of the two notices the cancellation first is pure timing, so both
// have to agree on the outcome.
func (s *ConsumerTestSuite) TestConsumerRunReturnsNoErrorWhenParentIsCanceledBetweenPolls() {
	t := s.T()
	reader := kafkaConsumerMocks.NewReader(t)
	record := &kgo.Record{Topic: "test-topic", Partition: 0, Offset: 1}
	ctx, cancel := context.WithCancel(t.Context())

	// A non empty fetch makes waitForRecords return immediately, so the loop reaches its cancellation check
	// with an already canceled context.
	reader.EXPECT().PollRecords(nil, 100).Return(fetchWithPartitionError([]*kgo.Record{record}, nil)).Once()
	reader.EXPECT().CommitRecords(matcher.Context, record).Return(nil).Once()
	reader.EXPECT().AllowRebalance().Once()
	reader.EXPECT().CloseAllowingRebalance().Once()
	s.metricWriter.EXPECT().Write(matcher.Context, mock.Anything).Times(3)

	readerFactory := func(_ context.Context, _ *consumer.PartitionManager) (consumer.Reader, error) {
		return reader, nil
	}
	s.clk = clock.NewRealClock()
	s.healthCheckTimer = clock.NewHealthCheckTimerWithInterfaces(s.clk, time.Minute)
	c := s.newTestConsumer(readerFactory, &consumer.Settings{MaxPollRecords: 100, GraceTime: time.Minute})

	errCh := make(chan error, 1)
	go func() {
		errCh <- c.Run(ctx, func(context.Context, *kgo.Record) bool {
			cancel()

			return true
		})
	}()

	select {
	case err := <-errCh:
		assert.NoError(t, err, "a canceled parent context must not be reported as a consumer error")
	case <-time.After(time.Second):
		t.Fatal("consumer did not stop after the parent context was canceled")
	}
}

// TestConsumerCommitCanceledByGraceTimeIsLogged guards the transparency of the one shutdown case that silently
// loses work: the records were processed but their offsets never reached the broker, so they are consumed
// again after a restart. runLoop deliberately swallows that error to keep the shutdown clean, which makes the
// log the only remaining signal.
func (s *ConsumerTestSuite) TestConsumerCommitCanceledByGraceTimeIsLogged() {
	t := s.T()
	reader := kafkaConsumerMocks.NewReader(t)
	record := &kgo.Record{Topic: "test-topic", Partition: 0, Offset: 1}
	commitStarted := make(chan struct{})
	handler := &recordingLogHandler{}

	reader.EXPECT().PollRecords(nil, 100).Return(fetchWithPartitionError([]*kgo.Record{record}, nil)).Once()
	reader.EXPECT().CommitRecords(matcher.Context, record).RunAndReturn(func(ctx context.Context, _ ...*kgo.Record) error {
		close(commitStarted)
		<-ctx.Done()

		return ctx.Err()
	}).Once()
	reader.EXPECT().CloseAllowingRebalance().Once()
	s.metricWriter.EXPECT().Write(matcher.Context, mock.Anything).Once()

	c := s.newTestConsumerWithLogHandler(
		func(_ context.Context, _ *consumer.PartitionManager) (consumer.Reader, error) {
			return reader, nil
		},
		&consumer.Settings{MaxPollRecords: 100, GraceTime: time.Millisecond},
		handler,
	)

	errCh := make(chan error, 1)
	go func() {
		errCh <- c.Run(t.Context(), func(context.Context, *kgo.Record) bool { return true })
	}()

	<-commitStarted
	c.Stop(t.Context())

	select {
	case err := <-errCh:
		assert.NoError(t, err, "a commit canceled during shutdown must not fail the consumer")
	case <-time.After(time.Second):
		t.Fatal("consumer did not stop after the commit grace time elapsed")
	}

	assert.True(t, handler.hasErrorContaining("1 records could not be committed and will be consumed again"),
		"expected a warning about the uncommitted records, got: %v", handler.messages())
}

// TestConsumeDelayWaitsUntilRecordIsOldEnough verifies that a record younger than the configured delay is held back
// until it reached that age, and that the completed wait is reported as SleepDuration.
func (s *ConsumerTestSuite) TestConsumeDelayWaitsUntilRecordIsOldEnough() {
	t := s.T()
	reader := kafkaConsumerMocks.NewReader(t)
	record := &kgo.Record{Topic: "test-topic", Partition: 0, Offset: 1, Timestamp: s.fakeClock.Now()}
	var processedAt []time.Time

	reader.EXPECT().PollRecords(nil, 100).Return(fetchWithPartitionError([]*kgo.Record{record}, nil)).Once()
	reader.EXPECT().PollRecords(nil, 100).Return(clientClosedFetches()).Once()
	reader.EXPECT().CommitRecords(matcher.Context, record).Return(nil).Once()
	reader.EXPECT().AllowRebalance().Once()
	reader.EXPECT().CloseAllowingRebalance().Once()
	sleepDurations := s.expectMetricWrites(4)

	c := s.newTestConsumer(func(_ context.Context, _ *consumer.PartitionManager) (consumer.Reader, error) {
		return reader, nil
	}, &consumer.Settings{MaxPollRecords: 100, RunnerCount: 1, GraceTime: time.Second, RebalanceTimeout: time.Minute, ConsumeDelay: time.Second})

	errCh := make(chan error, 1)
	go func() {
		errCh <- c.Run(t.Context(), func(context.Context, *kgo.Record) bool {
			processedAt = append(processedAt, s.fakeClock.Now())

			return true
		})
	}()

	s.fakeClock.BlockUntilTimers(1)
	s.fakeClock.Advance(time.Second)

	require.NoError(t, <-errCh)
	require.Len(t, processedAt, 1)
	assert.Equal(t, record.Timestamp.Add(time.Second), processedAt[0], "the record must not be handed to the callback before it reached the configured age")
	assert.Equal(t, []float64{1000}, sleepDurations())
}

// TestConsumeDelaySkipsRecordOlderThanTheDelay verifies that a record which already exceeds the delay is passed on
// without waiting, so a consumer catching up on a backlog is not slowed down.
func (s *ConsumerTestSuite) TestConsumeDelaySkipsRecordOlderThanTheDelay() {
	t := s.T()
	reader := kafkaConsumerMocks.NewReader(t)
	record := &kgo.Record{Topic: "test-topic", Partition: 0, Offset: 1, Timestamp: s.fakeClock.Now().Add(-time.Minute)}

	reader.EXPECT().PollRecords(nil, 100).Return(fetchWithPartitionError([]*kgo.Record{record}, nil)).Once()
	reader.EXPECT().PollRecords(nil, 100).Return(clientClosedFetches()).Once()
	reader.EXPECT().CommitRecords(matcher.Context, record).Return(nil).Once()
	reader.EXPECT().AllowRebalance().Once()
	reader.EXPECT().CloseAllowingRebalance().Once()
	sleepDurations := s.expectMetricWrites(3)

	c := s.newTestConsumer(func(_ context.Context, _ *consumer.PartitionManager) (consumer.Reader, error) {
		return reader, nil
	}, &consumer.Settings{MaxPollRecords: 100, RunnerCount: 1, GraceTime: time.Second, RebalanceTimeout: time.Minute, ConsumeDelay: time.Second})

	// no clock is advanced: the run only completes if the delay did not wait at all
	require.NoError(t, c.Run(t.Context(), func(context.Context, *kgo.Record) bool { return true }))
	assert.Empty(t, sleepDurations(), "an old record must not report a wait")
}

// TestConsumeDelayClampsRecordTimestampedIntoTheFuture guards against a producer clock running ahead of ours: with the
// default topic setting message.timestamp.type=CreateTime the timestamp comes from the producer, and a record dated
// into the future must not be held back for longer than the configured delay.
func (s *ConsumerTestSuite) TestConsumeDelayClampsRecordTimestampedIntoTheFuture() {
	t := s.T()
	reader := kafkaConsumerMocks.NewReader(t)
	record := &kgo.Record{Topic: "test-topic", Partition: 0, Offset: 1, Timestamp: s.fakeClock.Now().Add(time.Hour)}

	reader.EXPECT().PollRecords(nil, 100).Return(fetchWithPartitionError([]*kgo.Record{record}, nil)).Once()
	reader.EXPECT().PollRecords(nil, 100).Return(clientClosedFetches()).Once()
	reader.EXPECT().CommitRecords(matcher.Context, record).Return(nil).Once()
	reader.EXPECT().AllowRebalance().Once()
	reader.EXPECT().CloseAllowingRebalance().Once()
	sleepDurations := s.expectMetricWrites(4)

	c := s.newTestConsumer(func(_ context.Context, _ *consumer.PartitionManager) (consumer.Reader, error) {
		return reader, nil
	}, &consumer.Settings{MaxPollRecords: 100, RunnerCount: 1, GraceTime: time.Second, RebalanceTimeout: time.Minute, ConsumeDelay: time.Second})

	errCh := make(chan error, 1)
	go func() {
		errCh <- c.Run(t.Context(), func(context.Context, *kgo.Record) bool { return true })
	}()

	s.fakeClock.BlockUntilTimers(1)
	s.fakeClock.Advance(time.Second)

	require.NoError(t, <-errCh)
	assert.Equal(t, []float64{1000}, sleepDurations(), "the wait must be capped at the configured delay")
}

// TestConsumeDelayCanceledDuringWaitStillProcessesFirstRecord documents the interaction with the offset gap invariant of
// processRecordWorkUnits: the first record of an admitted work unit is committed, so a shutdown must not skip it. Its
// delay is cut short instead, which releases the record ahead of time but never commits it unprocessed.
func (s *ConsumerTestSuite) TestConsumeDelayCanceledDuringWaitStillProcessesFirstRecord() {
	t := s.T()
	reader := kafkaConsumerMocks.NewReader(t)
	record := &kgo.Record{Topic: "test-topic", Partition: 0, Offset: 1, Timestamp: s.fakeClock.Now()}
	var processed atomic.Int32

	reader.EXPECT().PollRecords(nil, 100).Return(fetchWithPartitionError([]*kgo.Record{record}, nil)).Once()
	reader.EXPECT().CommitRecords(matcher.Context, record).Return(nil).Once()
	reader.EXPECT().AllowRebalance().Once()
	reader.EXPECT().CloseAllowingRebalance().Once()
	sleepDurations := s.expectMetricWrites(3)

	c := s.newTestConsumer(func(_ context.Context, _ *consumer.PartitionManager) (consumer.Reader, error) {
		return reader, nil
	}, &consumer.Settings{MaxPollRecords: 100, RunnerCount: 1, GraceTime: time.Second, RebalanceTimeout: time.Minute, ConsumeDelay: time.Minute})

	errCh := make(chan error, 1)
	go func() {
		errCh <- c.Run(t.Context(), func(context.Context, *kgo.Record) bool {
			processed.Add(1)

			return true
		})
	}()

	// the timer only exists once we are actually waiting out the delay
	s.fakeClock.BlockUntilTimers(1)
	c.Stop(t.Context())

	select {
	case err := <-errCh:
		assert.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("consumer did not stop while waiting out the consume delay")
	}

	assert.Equal(t, int32(1), processed.Load(), "the first record of an admitted work unit must be processed even if its delay was cut short")
	assert.Empty(t, sleepDurations(), "an interrupted wait must not be reported")
}

// TestConsumeDelayCanceledDuringWaitStopsAtRecordBoundary verifies the other half of the invariant: a record which is
// not the first of its work unit is left for redelivery when the shutdown interrupts its delay, so only the records
// processed before it are committed.
func (s *ConsumerTestSuite) TestConsumeDelayCanceledDuringWaitStopsAtRecordBoundary() {
	t := s.T()
	reader := kafkaConsumerMocks.NewReader(t)
	first := &kgo.Record{Topic: "test-topic", Partition: 0, Offset: 1, Timestamp: s.fakeClock.Now().Add(-time.Minute)}
	second := &kgo.Record{Topic: "test-topic", Partition: 0, Offset: 2, Timestamp: s.fakeClock.Now()}
	var processed []int64
	var processedLck sync.Mutex

	reader.EXPECT().PollRecords(nil, 100).Return(fetchWithPartitionError([]*kgo.Record{first, second}, nil)).Once()
	reader.EXPECT().CommitRecords(matcher.Context, first).Return(nil).Once()
	reader.EXPECT().AllowRebalance().Once()
	reader.EXPECT().CloseAllowingRebalance().Once()
	s.expectMetricWrites(3)

	c := s.newTestConsumer(func(_ context.Context, _ *consumer.PartitionManager) (consumer.Reader, error) {
		return reader, nil
	}, &consumer.Settings{
		MaxPollRecords: 100, RunnerCount: 1, GraceTime: time.Second, RebalanceTimeout: time.Minute,
		ConsumeDelay: time.Minute, ProcessingMode: consumer.ProcessingModeOrdered,
	})

	errCh := make(chan error, 1)
	go func() {
		errCh <- c.Run(t.Context(), func(_ context.Context, record *kgo.Record) bool {
			processedLck.Lock()
			defer processedLck.Unlock()
			processed = append(processed, record.Offset)

			return true
		})
	}()

	// the first record is old enough to pass through, so the only timer belongs to the second one
	s.fakeClock.BlockUntilTimers(1)
	c.Stop(t.Context())

	select {
	case err := <-errCh:
		assert.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("consumer did not stop while waiting out the consume delay")
	}

	processedLck.Lock()
	defer processedLck.Unlock()
	assert.Equal(t, []int64{1}, processed, "the record whose delay was interrupted must be left for redelivery")
}

// TestConsumeDelayKeepsConsumerHealthy verifies that waiting out a delay longer than the health check timeout does not
// turn the consumer unhealthy: being killed for waiting exactly as configured would make the setting unusable.
func (s *ConsumerTestSuite) TestConsumeDelayKeepsConsumerHealthy() {
	t := s.T()
	healthCheckTimeout := 2 * time.Second
	s.healthCheckTimer = clock.NewHealthCheckTimerWithInterfaces(s.fakeClock, healthCheckTimeout)

	reader := kafkaConsumerMocks.NewReader(t)
	record := &kgo.Record{Topic: "test-topic", Partition: 0, Offset: 1, Timestamp: s.fakeClock.Now()}

	reader.EXPECT().PollRecords(nil, 100).Return(fetchWithPartitionError([]*kgo.Record{record}, nil)).Once()
	reader.EXPECT().PollRecords(nil, 100).Return(clientClosedFetches()).Once()
	reader.EXPECT().CommitRecords(matcher.Context, record).Return(nil).Once()
	reader.EXPECT().AllowRebalance().Once()
	reader.EXPECT().CloseAllowingRebalance().Once()
	sleepDurations := s.expectMetricWrites(4)

	c := s.newTestConsumer(func(_ context.Context, _ *consumer.PartitionManager) (consumer.Reader, error) {
		return reader, nil
	}, &consumer.Settings{
		MaxPollRecords: 100, RunnerCount: 1, GraceTime: time.Second, RebalanceTimeout: 5 * time.Minute,
		ConsumeDelay: 30 * time.Second, Healthcheck: health.HealthCheckSettings{Timeout: healthCheckTimeout},
	})

	errCh := make(chan error, 1)
	go func() {
		errCh <- c.Run(t.Context(), func(context.Context, *kgo.Record) bool { return true })
	}()

	// wait for the keep alive ticker, which delayConsume creates after the timer for the delay itself
	s.fakeClock.BlockUntilTickers(1)
	s.fakeClock.Advance(3 * healthCheckTimeout / 2)

	assert.Eventually(t, c.IsHealthy, time.Second, time.Millisecond,
		"the consumer must stay healthy while it waits out the consume delay")

	s.fakeClock.Advance(30 * time.Second)

	require.NoError(t, <-errCh)
	assert.Equal(t, []float64{30_000}, sleepDurations())
}

// expectMetricWrites accepts exactly the given number of metric writes and returns an accessor for the SleepDuration
// values among them. The count is asserted because an unexpected write means the delay reported a wait it should not
// have, or reported none where it should have.
func (s *ConsumerTestSuite) expectMetricWrites(count int) func() []float64 {
	var lck sync.Mutex
	var sleepDurations []float64

	s.metricWriter.EXPECT().Write(matcher.Context, mock.Anything).Run(func(_ context.Context, data metric.Data) {
		lck.Lock()
		defer lck.Unlock()

		for _, datum := range data {
			if datum.MetricName == "SleepDuration" {
				sleepDurations = append(sleepDurations, datum.Value)
			}
		}
	}).Times(count)

	return func() []float64 {
		lck.Lock()
		defer lck.Unlock()

		return sleepDurations
	}
}

// recordingLogHandler collects the messages a logger emits so a test can assert on them. A mocked logger is
// not usable here because the backoff executor derives its own logger via WithFields.
type recordingLogHandler struct {
	lck     sync.Mutex
	entries []recordedLogEntry
}

type recordedLogEntry struct {
	level int
	msg   string
}

func (h *recordingLogHandler) ChannelLevel(string) (*int, error) {
	return nil, nil
}

func (h *recordingLogHandler) Level() int {
	return log.PriorityDebug
}

func (h *recordingLogHandler) Log(_ context.Context, _ time.Time, level int, msg string, args []any, _ error, _ log.Data) error {
	h.lck.Lock()
	defer h.lck.Unlock()

	h.entries = append(h.entries, recordedLogEntry{level: level, msg: fmt.Sprintf(msg, args...)})

	return nil
}

func (h *recordingLogHandler) hasErrorContaining(needle string) bool {
	h.lck.Lock()
	defer h.lck.Unlock()

	for _, entry := range h.entries {
		if entry.level == log.PriorityError && strings.Contains(entry.msg, needle) {
			return true
		}
	}

	return false
}

func (h *recordingLogHandler) messages() []string {
	h.lck.Lock()
	defer h.lck.Unlock()

	msgs := make([]string, 0, len(h.entries))
	for _, entry := range h.entries {
		msgs = append(msgs, entry.msg)
	}

	return msgs
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

func fetchWithPartitions(records ...*kgo.Record) kgo.Fetches {
	partitions := make([]kgo.FetchPartition, 0, len(records))
	for _, record := range records {
		partitions = append(partitions, kgo.FetchPartition{
			Partition: record.Partition,
			Records:   []*kgo.Record{record},
		})
	}

	return kgo.Fetches{{
		Topics: []kgo.FetchTopic{{
			Topic:      "test-topic",
			Partitions: partitions,
		}},
	}}
}

func fetchWithDuplicatePartition(records ...*kgo.Record) kgo.Fetches {
	return kgo.Fetches{
		{
			Topics: []kgo.FetchTopic{
				{
					Topic: "test-topic",
					Partitions: []kgo.FetchPartition{
						{Partition: 0, Records: []*kgo.Record{records[0]}},
					},
				},
			},
		},
		{
			Topics: []kgo.FetchTopic{
				{
					Topic: "test-topic",
					Partitions: []kgo.FetchPartition{
						{Partition: 0, Records: []*kgo.Record{records[1]}},
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
