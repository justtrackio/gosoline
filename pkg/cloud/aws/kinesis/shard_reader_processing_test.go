package kinesis

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kinesis/types"
	"github.com/justtrackio/gosoline/pkg/exec"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/metric"
	metricMocks "github.com/justtrackio/gosoline/pkg/metric/mocks"
	"github.com/justtrackio/gosoline/pkg/test/matcher"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestShardReaderProcessesRecordsSeriallyInOrderedMode(t *testing.T) {
	checkpoint := &recordingCheckpoint{}
	reader := newProcessingTestShardReader(t, ProcessingModeOrdered, 2, checkpoint)
	firstStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	var callsMu sync.Mutex
	var calls []string

	done := make(chan error, 1)
	go func() {
		lastSequenceNumber := SequenceNumber("")
		_, err := reader.processRecords(t.Context(), t.Context(), processingTestRecords(), &lastSequenceNumber, "", func(_ context.Context, data []byte) error {
			callsMu.Lock()
			calls = append(calls, string(data))
			callsMu.Unlock()

			if string(data) == "first" {
				close(firstStarted)
				<-releaseFirst
			}

			return nil
		})
		done <- err
	}()

	<-firstStarted
	callsMu.Lock()
	require.Equal(t, []string{"first"}, calls)
	callsMu.Unlock()
	close(releaseFirst)
	require.NoError(t, <-done)

	callsMu.Lock()
	require.Equal(t, []string{"first", "second"}, calls)
	callsMu.Unlock()
	require.Equal(t, []SequenceNumber{"seq-1", "seq-2"}, checkpoint.sequenceNumbers())
}

func TestShardReaderProcessesRecordsConcurrentlyAndCheckpointsInOrderInUnorderedMode(t *testing.T) {
	checkpoint := &recordingCheckpoint{}
	reader := newProcessingTestShardReader(t, ProcessingModeUnordered, 2, checkpoint)
	firstStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	secondFinished := make(chan struct{})

	done := make(chan error, 1)
	go func() {
		lastSequenceNumber := SequenceNumber("")
		_, err := reader.processRecords(t.Context(), t.Context(), processingTestRecords(), &lastSequenceNumber, "", func(_ context.Context, data []byte) error {
			switch string(data) {
			case "first":
				close(firstStarted)
				<-releaseFirst
			case "second":
				close(secondFinished)
			}

			return nil
		})
		done <- err
	}()

	<-firstStarted
	<-secondFinished
	require.Empty(t, checkpoint.sequenceNumbers())
	close(releaseFirst)
	require.NoError(t, <-done)
	require.Equal(t, []SequenceNumber{"seq-1", "seq-2"}, checkpoint.sequenceNumbers())
}

func TestShardReaderDoesNotCheckpointRecordsSkippedDuringCancellation(t *testing.T) {
	checkpoint := &recordingCheckpoint{}
	reader := newProcessingTestShardReader(t, ProcessingModeUnordered, 2, checkpoint)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	lastSequenceNumber := SequenceNumber("")

	processed, err := reader.processRecords(ctx, t.Context(), processingTestRecords(), &lastSequenceNumber, "", func(_ context.Context, _ []byte) error {
		t.Fatal("handler should not be called")

		return nil
	})

	require.NoError(t, err)
	require.Zero(t, processed)
	require.Empty(t, checkpoint.sequenceNumbers())
}

func TestShardReaderDrainsInFlightRecordsAndSkipsQueuedRecordsDuringShutdown(t *testing.T) {
	checkpoint := &recordingCheckpoint{}
	reader := newProcessingTestShardReader(t, ProcessingModeUnordered, 2, checkpoint)
	ctx, cancel := context.WithCancel(t.Context())
	processingCtx, cancelProcessing := context.WithCancel(context.WithoutCancel(ctx))
	defer cancelProcessing()
	entered := make(chan struct{}, 2)
	handlerContexts := make(chan error, 2)
	release := make(chan struct{})
	var calls atomic.Int32

	done := make(chan struct{})
	var processed int
	var processErr error
	go func() {
		defer close(done)

		lastSequenceNumber := SequenceNumber("")
		processed, processErr = reader.processRecords(ctx, processingCtx, []types.Record{
			{Data: []byte("first"), SequenceNumber: aws.String("seq-1")},
			{Data: []byte("second"), SequenceNumber: aws.String("seq-2")},
			{Data: []byte("third"), SequenceNumber: aws.String("seq-3")},
			{Data: []byte("fourth"), SequenceNumber: aws.String("seq-4")},
		}, &lastSequenceNumber, "", func(handlerCtx context.Context, _ []byte) error {
			calls.Add(1)
			entered <- struct{}{}
			handlerContexts <- handlerCtx.Err()
			<-release

			return nil
		})
	}()

	<-entered
	<-entered
	cancel()
	close(release)
	<-done

	require.NoError(t, processErr)
	require.NoError(t, <-handlerContexts)
	require.NoError(t, <-handlerContexts)
	require.Equal(t, int32(2), calls.Load())
	require.Equal(t, 2, processed)
	require.Equal(t, []SequenceNumber{"seq-1", "seq-2"}, checkpoint.sequenceNumbers())
}

func TestShardReaderCancelsProcessingContextAtProcessingDeadline(t *testing.T) {
	checkpoint := &recordingCheckpoint{}
	reader := newProcessingTestShardReader(t, ProcessingModeUnordered, 1, checkpoint)
	reader.logger.(logMocks.LoggerMock).EXPECT().Warn(
		matcher.Context,
		"processing deadline expired, cancelling in-flight record processing",
	).Once()

	drainCtx, expireProcessingDeadline := context.WithCancel(t.Context())
	defer expireProcessingDeadline()

	ctx, cancel := context.WithCancel(exec.WithDrainContext(t.Context(), drainCtx))
	processingCtx, stop := reader.drainContext(ctx)
	defer stop()

	// Cancelling the shard itself must not reach an admitted handler: only the caller decides when processing ends.
	cancel()
	require.NoError(t, processingCtx.Err())

	expireProcessingDeadline()
	<-processingCtx.Done()
	require.ErrorIs(t, processingCtx.Err(), context.Canceled)
}

// TestShardReaderPropagatesCancellationWithoutProcessingDeadline guards the contract for callers which run a shard
// reader without attaching a drain context: they own cancellation themselves, so it must reach handlers immediately
// rather than being delayed by a grace period the shard reader invented on their behalf.
func TestShardReaderPropagatesCancellationWithoutProcessingDeadline(t *testing.T) {
	checkpoint := &recordingCheckpoint{}
	reader := newProcessingTestShardReader(t, ProcessingModeUnordered, 1, checkpoint)

	ctx, cancel := context.WithCancel(t.Context())
	processingCtx, stop := reader.drainContext(ctx)
	defer stop()

	cancel()
	<-processingCtx.Done()
	require.ErrorIs(t, processingCtx.Err(), context.Canceled)
}

// TestShardReaderCheckpointsRecordHandledUnderCanceledContext guards against a record being consumed again although
// its handler already dealt with it: a handler which completes its work is authoritative, no matter whether the
// processing context expired in the meantime. Reporting a cancellation here would stop the checkpoint before that
// record while a stream consumer has already handed it to the retry queue, delivering it twice.
func TestShardReaderCheckpointsRecordHandledUnderCanceledContext(t *testing.T) {
	checkpoint := &recordingCheckpoint{}
	reader := newProcessingTestShardReader(t, ProcessingModeOrdered, 1, checkpoint)
	processingCtx, cancelProcessing := context.WithCancel(t.Context())

	lastSequenceNumber := SequenceNumber("")
	processed, err := reader.processRecords(t.Context(), processingCtx, processingTestRecords(), &lastSequenceNumber, "", func(_ context.Context, data []byte) error {
		if string(data) == "first" {
			cancelProcessing()
		}

		return nil
	})

	require.NoError(t, err)
	require.Equal(t, 2, processed)
	require.Equal(t, []SequenceNumber{"seq-1", "seq-2"}, checkpoint.sequenceNumbers())
}

func TestShardReaderDoesNotCheckpointRecordAbandonedByCanceledHandler(t *testing.T) {
	checkpoint := &recordingCheckpoint{}
	reader := newProcessingTestShardReader(t, ProcessingModeUnordered, 2, checkpoint)
	processingCtx, cancelProcessing := context.WithCancel(t.Context())
	firstStarted := make(chan struct{})
	secondFinished := make(chan struct{})

	lastSequenceNumber := SequenceNumber("")
	done := make(chan struct{})
	var processed int
	var processErr error
	go func() {
		defer close(done)

		processed, processErr = reader.processRecords(t.Context(), processingCtx, processingTestRecords(), &lastSequenceNumber, "", func(ctx context.Context, data []byte) error {
			if string(data) == "first" {
				close(firstStarted)
				<-ctx.Done()

				return ctx.Err()
			}

			close(secondFinished)

			return nil
		})
	}()

	<-firstStarted
	<-secondFinished
	cancelProcessing()
	<-done

	require.NoError(t, processErr)
	require.Zero(t, processed)
	require.Empty(t, checkpoint.sequenceNumbers())
}

func TestShardReaderRecoversHandlerPanicInOrderedMode(t *testing.T) {
	testShardReaderRecoversHandlerPanic(t, ProcessingModeOrdered)
}

func TestShardReaderRecoversHandlerPanicInUnorderedMode(t *testing.T) {
	testShardReaderRecoversHandlerPanic(t, ProcessingModeUnordered)
}

func testShardReaderRecoversHandlerPanic(t *testing.T, mode ProcessingMode) {
	t.Helper()

	checkpoint := &recordingCheckpoint{}
	reader := newProcessingTestShardReader(t, mode, 2, checkpoint)
	logger := reader.logger.(logMocks.LoggerMock)
	metricWriter := reader.metricWriter.(*metricMocks.Writer)
	logger.EXPECT().Error(matcher.Context, "failed to handle record %s: %w", "seq-1", mock.Anything).Once()
	metricWriter.EXPECT().Write(matcher.Context, metric.Data{
		{
			Priority:   metric.PriorityHigh,
			MetricName: metricNameFailedRecords,
			Dimensions: metric.Dimensions{"StreamName": ""},
			Value:      1,
			Unit:       metric.UnitCount,
			Kind:       metric.KindTotal,
		},
		{
			Priority:   metric.PriorityHigh,
			MetricName: metricNameFailedRecords,
			Dimensions: metric.Dimensions{"StreamName": "", "ShardId": ""},
			Value:      1,
			Unit:       metric.UnitCount,
		},
	}).Once()

	lastSequenceNumber := SequenceNumber("")
	processed, err := reader.processRecords(t.Context(), t.Context(), processingTestRecords(), &lastSequenceNumber, "", func(_ context.Context, data []byte) error {
		if string(data) == "first" {
			panic("handler panic")
		}

		return nil
	})

	require.NoError(t, err)
	require.Equal(t, len(processingTestRecords()), processed)
	require.Equal(t, []SequenceNumber{"seq-1", "seq-2"}, checkpoint.sequenceNumbers())
}

type recordingCheckpoint struct {
	mutex    sync.Mutex
	advances []SequenceNumber
}

func (c *recordingCheckpoint) GetSequenceNumber() SequenceNumber {
	return ""
}

func (c *recordingCheckpoint) GetShardIterator() ShardIterator {
	return ""
}

func (c *recordingCheckpoint) Advance(sequenceNumber SequenceNumber, _ ShardIterator) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.advances = append(c.advances, sequenceNumber)

	return nil
}

func (c *recordingCheckpoint) Done(_ SequenceNumber) error {
	return nil
}

func (c *recordingCheckpoint) Persist(_ context.Context) (bool, error) {
	return false, nil
}

func (c *recordingCheckpoint) Release(_ context.Context) error {
	return nil
}

func (c *recordingCheckpoint) sequenceNumbers() []SequenceNumber {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	return append([]SequenceNumber(nil), c.advances...)
}

type recordingHealthCheckTimer struct{}

func (recordingHealthCheckTimer) IsHealthy() bool {
	return true
}

func (recordingHealthCheckTimer) MarkHealthy() {}

func newProcessingTestShardReader(t *testing.T, mode ProcessingMode, runnerCount int, checkpoint Checkpoint) *shardReader {
	reader := &shardReader{
		logger:       logMocks.NewLoggerMock(logMocks.WithTestingT(t)),
		metricWriter: metricMocks.NewWriter(t),
		settings: Settings{
			ProcessingMode: mode,
			RunnerCount:    runnerCount,
		},
		healthCheckTimer: recordingHealthCheckTimer{},
	}
	reader.checkpoint.Store(checkpointWrapper{Checkpoint: checkpoint})

	return reader
}

func processingTestRecords() []types.Record {
	return []types.Record{
		{Data: []byte("first"), SequenceNumber: aws.String("seq-1")},
		{Data: []byte("second"), SequenceNumber: aws.String("seq-2")},
	}
}
