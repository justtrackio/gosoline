package kinesis

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	cloudwatchtypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	awsKinesis "github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/aws/aws-sdk-go-v2/service/kinesis/types"
	"github.com/justtrackio/gosoline/pkg/clock"
	cloudwatchMocks "github.com/justtrackio/gosoline/pkg/cloud/aws/cloudwatch/mocks"
	"github.com/justtrackio/gosoline/pkg/exec"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/metric"
	metricMocks "github.com/justtrackio/gosoline/pkg/metric/mocks"
	"github.com/justtrackio/gosoline/pkg/test/matcher"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
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
			Namespace:  metricNamespaceCloudAwsKinesis,
			MetricName: metricNameConsumedMessages,
			Dimensions: metric.Dimensions{
				dimensionStream:           "",
				dimensionShard:            "",
				metric.DimensionErrorType: errorTypeProcessingFailed,
			},
			Value: 1,
			Unit:  metric.UnitCount,
			Kind:  metric.KindCounter.Build(),
		},
		{
			Priority:   metric.PriorityHigh,
			Namespace:  metricNamespaceCloudAwsKinesis,
			MetricName: metricNameConsumedMessages,
			Dimensions: metric.Dimensions{
				dimensionStream:           "",
				metric.DimensionErrorType: errorTypeProcessingFailed,
			},
			Value: 1,
			Unit:  metric.UnitCount,
			Kind:  metric.KindTotal,
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

func TestShardReaderDefaultMetricsSeedOnlyMissingStreamProjection(t *testing.T) {
	defaults := getShardReaderDefaultMetrics(Stream("events"))

	require.ElementsMatch(t, metric.Data{
		{
			Priority:   metric.PriorityHigh,
			MetricName: metricNameReadCount,
			Dimensions: metric.Dimensions{
				dimensionStream: "events",
				dimensionShard:  metric.DimensionDefault,
			},
			Unit:  metric.UnitCount,
			Value: 0,
			Kind:  metric.KindCounter.Build(),
		},
		{
			Priority:   metric.PriorityHigh,
			MetricName: metricNameConsumedMessages,
			Dimensions: metric.Dimensions{
				dimensionStream:           "events",
				dimensionShard:            metric.DimensionDefault,
				metric.DimensionErrorType: metric.DimensionDefault,
			},
			Unit:  metric.UnitCount,
			Value: 0,
			Kind:  metric.KindCounter.Build(),
		},
		{
			Priority:   metric.PriorityHigh,
			MetricName: metricNameConsumedMessages,
			Dimensions: metric.Dimensions{
				dimensionStream:           "events",
				metric.DimensionErrorType: errorTypeProcessingFailed,
			},
			Unit:  metric.UnitCount,
			Value: 0,
			Kind:  metric.KindTotal,
		},
	}, defaults)
	for _, datum := range defaults {
		datum.Namespace = metricNamespaceCloudAwsKinesis
	}

	now := time.Unix(1549283566, 0)
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))
	cloudwatchClient := cloudwatchMocks.NewClient(t)
	var cloudwatchData []cloudwatchtypes.MetricDatum
	cloudwatchClient.EXPECT().PutMetricData(matcher.Context, mock.Anything).
		Run(func(_ context.Context, input *cloudwatch.PutMetricDataInput, _ ...func(*cloudwatch.Options)) {
			require.Equal(t, "test", aws.ToString(input.Namespace))
			cloudwatchData = append(cloudwatchData, input.MetricData...)
		}).
		Return(nil, nil).
		Once()
	writer := metric.NewCloudwatchWriterWithInterfaces(
		logger,
		clock.NewFakeClockAt(now),
		cloudwatchClient,
		"test",
		time.Second,
	)
	writer.Write(t.Context(), defaults)

	require.Len(t, cloudwatchData, 3)
	seen := map[string]bool{}
	for _, datum := range cloudwatchData {
		require.Equal(t, now, aws.ToTime(datum.Timestamp))
		require.Equal(t, metric.UnitCount, datum.Unit)
		require.Equal(t, 0.0, aws.ToFloat64(datum.Value))
		dimensions := make(metric.Dimensions, len(datum.Dimensions))
		for _, dimension := range datum.Dimensions {
			dimensions[aws.ToString(dimension.Name)] = aws.ToString(dimension.Value)
		}

		switch aws.ToString(datum.MetricName) {
		case "CloudAwsKinesisReads":
			require.Equal(t, metric.Dimensions{dimensionStream: "events"}, dimensions)
			require.False(t, seen["reads"])
			seen["reads"] = true
		case "CloudAwsKinesisConsumedMessages":
			errorType := dimensions[metric.DimensionErrorType]
			switch errorType {
			case "":
				require.Equal(t, metric.Dimensions{dimensionStream: "events"}, dimensions)
				require.False(t, seen["consumed_total"])
				seen["consumed_total"] = true
			case errorTypeProcessingFailed:
				require.Equal(t, metric.Dimensions{
					dimensionStream:           "events",
					metric.DimensionErrorType: errorTypeProcessingFailed,
				}, dimensions)
				require.False(t, seen["consumed_failed"])
				seen["consumed_failed"] = true
			default:
				t.Fatalf("unexpected consumed.messages error.type %q", errorType)
			}
		default:
			t.Fatalf("unexpected CloudWatch metric %q", aws.ToString(datum.MetricName))
		}
	}
	require.Equal(t, map[string]bool{
		"reads":           true,
		"consumed_total":  true,
		"consumed_failed": true,
	}, seen)
}

func TestShardReaderEmitsStandardProcessDurationHistogram(t *testing.T) {
	checkpoint := &recordingCheckpoint{}
	reader := newProcessingTestShardReader(t, ProcessingModeOrdered, 1, checkpoint)
	reader.stream = Stream("events")
	reader.shardId = ShardId("shard-1")
	reader.settings.MaxBatchSize = 1
	reader.settings.WaitTime = 0

	testClock := clock.NewFakeClockAt(time.Unix(0, 0))
	reader.clock = testClock
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))
	reader.logger = logger

	exportReader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(exportReader))
	t.Cleanup(func() {
		require.NoError(t, provider.Shutdown(t.Context()))
	})

	exportingWriter := metric.NewOtelWriterWithInterfaces(logger, provider.Meter("kinesis-process-duration-test"))
	reader.metricWriter = exportingWriter
	reader.kinesisClient = getRecordsTestClient{
		output: &awsKinesis.GetRecordsOutput{
			Records: []types.Record{
				{Data: []byte("record"), SequenceNumber: aws.String("seq-1")},
			},
			NextShardIterator:  aws.String("next"),
			MillisBehindLatest: aws.Int64(0),
		},
	}

	millisecondsBehind := make(chan float64, 1)
	iterator := ShardIterator("iterator")
	lastSequenceNumber := SequenceNumber("")
	_, err := reader.getAndProcessRecords(
		t.Context(),
		t.Context(),
		millisecondsBehind,
		&iterator,
		&lastSequenceNumber,
		func(context.Context, []byte) error {
			testClock.Advance(250 * time.Millisecond)

			return nil
		},
	)
	require.NoError(t, err)
	require.Equal(t, 0.0, <-millisecondsBehind)
	require.Equal(t, []SequenceNumber{"seq-1"}, checkpoint.sequenceNumbers())

	var exported metricdata.ResourceMetrics
	require.NoError(t, exportReader.Collect(t.Context(), &exported))

	var processHistogram *metricdata.Histogram[float64]
	for _, scope := range exported.ScopeMetrics {
		for i := range scope.Metrics {
			exportedMetric := &scope.Metrics[i]
			if exportedMetric.Name != "gosoline.cloud.aws.kinesis.process.duration" {
				continue
			}

			require.Nil(t, processHistogram, "process.duration must be exported as one histogram instrument")
			require.Equal(t, "s", exportedMetric.Unit)
			histogram, ok := exportedMetric.Data.(metricdata.Histogram[float64])
			require.True(t, ok)
			processHistogram = &histogram
		}
	}

	require.NotNil(t, processHistogram)
	require.Len(t, processHistogram.DataPoints, 1, "the stream-only KindTotal projection must be skipped by OTEL")
	point := processHistogram.DataPoints[0]
	require.Equal(t, []float64{0.005, 0.01, 0.025, 0.05, 0.075, 0.1, 0.25, 0.5, 0.75, 1, 2.5, 5, 7.5, 10}, point.Bounds)
	require.Equal(t, uint64(1), point.Count)
	require.InDelta(t, 0.25, point.Sum, 1e-12)

	dimensions := make(metric.Dimensions, len(point.Attributes.ToSlice()))
	for _, attribute := range point.Attributes.ToSlice() {
		dimensions[string(attribute.Key)] = attribute.Value.AsString()
	}
	require.Equal(t, metric.Dimensions{dimensionStream: "events", dimensionShard: "shard-1"}, dimensions)
}

type getRecordsTestClient struct {
	Client
	output *awsKinesis.GetRecordsOutput
}

func (c getRecordsTestClient) GetRecords(_ context.Context, _ *awsKinesis.GetRecordsInput, _ ...func(*awsKinesis.Options)) (*awsKinesis.GetRecordsOutput, error) {
	return c.output, nil
}
