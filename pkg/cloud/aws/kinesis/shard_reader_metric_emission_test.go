package kinesis

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsKinesis "github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/aws/aws-sdk-go-v2/service/kinesis/types"
	"github.com/justtrackio/gosoline/pkg/clock"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/metric"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestShardReaderEmitsBatchTotalAndFailureSubsetToOtel(t *testing.T) {
	ctx := t.Context()
	testClock := clock.NewFakeClock()
	batchStart := testClock.Now()
	otelReader, metricWriter := newKinesisOtelMetricWriter(t, testClock)

	records := make([]types.Record, 100)
	for i := range records {
		records[i] = types.Record{
			Data:           []byte("record"),
			SequenceNumber: aws.String(fmt.Sprintf("seq-%d", i+1)),
		}
	}

	kinesisClient := &kinesisRecordsClient{
		output: &awsKinesis.GetRecordsOutput{
			Records:            records,
			MillisBehindLatest: aws.Int64(0),
			NextShardIterator:  aws.String("next iterator"),
		},
	}

	checkpoint := &recordingCheckpoint{}
	reader := &shardReader{
		stream:           "test-stream",
		shardId:          "shard-001",
		logger:           logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t)),
		metricWriter:     metricWriter,
		kinesisClient:    kinesisClient,
		settings:         Settings{ProcessingMode: ProcessingModeOrdered, MaxBatchSize: len(records), WaitTime: 10 * time.Minute},
		clock:            testClock,
		healthCheckTimer: recordingHealthCheckTimer{},
	}
	reader.checkpoint.Store(checkpointWrapper{Checkpoint: checkpoint})

	iterator := ShardIterator("iterator")
	lastSequenceNumber := SequenceNumber("")
	millisecondsBehind := make(chan float64, 1)
	processed := 0

	_, err := reader.getAndProcessRecords(ctx, ctx, millisecondsBehind, &iterator, &lastSequenceNumber, func(context.Context, []byte) error {
		if processed == 0 {
			testClock.Advance(2 * time.Minute)
		} else if processed == len(records)-1 {
			testClock.Advance(3 * time.Minute)
		}

		processed++
		if processed <= 10 {
			return errors.New("handler failed")
		}

		return nil
	})
	require.NoError(t, err)
	require.Equal(t, 1, kinesisClient.calls)
	require.Equal(t, &awsKinesis.GetRecordsInput{ShardIterator: aws.String("iterator"), Limit: aws.Int32(100)}, kinesisClient.lastInput)
	require.Equal(t, len(records), processed)
	require.Equal(t, ShardIterator("next iterator"), iterator)
	require.Equal(t, SequenceNumber("seq-100"), lastSequenceNumber)
	require.Len(t, checkpoint.sequenceNumbers(), len(records))

	assertKinesisBatchEmissions(t, metricWriter.emissions(), batchStart)

	var exported metricdata.ResourceMetrics
	require.NoError(t, otelReader.Collect(ctx, &exported))

	var emittedNames []string
	for _, scope := range exported.ScopeMetrics {
		for _, emitted := range scope.Metrics {
			emittedNames = append(emittedNames, emitted.Name)
		}
	}
	require.ElementsMatch(t, []string{
		"gosoline.cloud.aws.kinesis.consumed.messages",
		"gosoline.cloud.aws.kinesis.process.duration",
		"gosoline.cloud.aws.kinesis.reads",
		"gosoline.cloud.aws.kinesis.wait.duration",
	}, emittedNames)
	consumed := kinesisOtelMetric(t, exported, "gosoline.cloud.aws.kinesis.consumed.messages")
	require.Equal(t, "{message}", consumed.Unit)
	counter, ok := consumed.Data.(metricdata.Sum[float64])
	require.True(t, ok)
	require.True(t, counter.IsMonotonic)
	require.Len(t, counter.DataPoints, 2)

	valuesByErrorType := make(map[string]float64, len(counter.DataPoints))
	for _, point := range counter.DataPoints {
		errorType, hasErrorType := kinesisOtelStringAttribute(point.Attributes, "error.type")
		require.Equal(t, "test-stream", kinesisOtelStringAttributeValue(t, point.Attributes, "stream.name"))
		require.Equal(t, "shard-001", kinesisOtelStringAttributeValue(t, point.Attributes, "partition.id"))
		if !hasErrorType {
			valuesByErrorType[metric.DimensionDefault] = point.Value

			continue
		}
		valuesByErrorType[errorType] = point.Value
	}
	require.Equal(t, map[string]float64{
		metric.DimensionDefault:   100,
		errorTypeProcessingFailed: 10,
	}, valuesByErrorType)

	readsMetric := kinesisOtelMetric(t, exported, "gosoline.cloud.aws.kinesis.reads")
	require.Equal(t, "{read}", readsMetric.Unit)
	readsCounter, ok := readsMetric.Data.(metricdata.Sum[float64])
	require.True(t, ok)
	require.Len(t, readsCounter.DataPoints, 1)
	require.Equal(t, float64(1), readsCounter.DataPoints[0].Value)
}

func TestShardReaderLagRetainsCumulativeOtelMaximum(t *testing.T) {
	testClock := clock.NewFakeClock()
	otelReader, metricWriter := newKinesisOtelMetricWriter(t, testClock)
	reader := &shardReader{
		stream:       "test-stream",
		shardId:      "shard-001",
		metricWriter: metricWriter,
		clock:        testClock,
	}

	millisecondsBehind := make(chan float64)
	done := make(chan struct{})
	go func() {
		reader.reportMillisecondsBehind(millisecondsBehind)
		close(done)
	}()

	waitForKinesisMetricWrites(t, metricWriter, 1)
	millisecondsBehind <- 9_000
	waitForKinesisMetricWrites(t, metricWriter, 1)

	peak := kinesisOtelMetric(t, collectKinesisOtelMetrics(t, otelReader), "gosoline.cloud.aws.kinesis.lag")
	require.Equal(t, "s", peak.Unit)
	peakHistogram, ok := peak.Data.(metricdata.Histogram[float64])
	require.True(t, ok)
	require.Equal(t, metricdata.CumulativeTemporality, peakHistogram.Temporality)
	require.Len(t, peakHistogram.DataPoints, 1)
	require.Equal(t, uint64(2), peakHistogram.DataPoints[0].Count)
	peakMax, hasPeakMax := peakHistogram.DataPoints[0].Max.Value()
	require.True(t, hasPeakMax)
	require.Equal(t, float64(9), peakMax)
	require.Equal(t, "test-stream", kinesisOtelStringAttributeValue(t, peakHistogram.DataPoints[0].Attributes, "stream.name"))
	require.Equal(t, "shard-001", kinesisOtelStringAttributeValue(t, peakHistogram.DataPoints[0].Attributes, "partition.id"))

	millisecondsBehind <- 1_000
	waitForKinesisMetricWrites(t, metricWriter, 1)

	fallen := kinesisOtelMetric(t, collectKinesisOtelMetrics(t, otelReader), "gosoline.cloud.aws.kinesis.lag")
	fallenHistogram, ok := fallen.Data.(metricdata.Histogram[float64])
	require.True(t, ok)
	require.Len(t, fallenHistogram.DataPoints, 1)
	require.Equal(t, uint64(3), fallenHistogram.DataPoints[0].Count)
	require.InDelta(t, 10, fallenHistogram.DataPoints[0].Sum, 1e-9)
	fallenMax, hasFallenMax := fallenHistogram.DataPoints[0].Max.Value()
	require.True(t, hasFallenMax)
	require.Equal(t, float64(9), fallenMax)

	var lagValues []float64
	for _, emission := range metricWriter.emissions() {
		for _, datum := range emission.data {
			if datum.MetricName != metricNameMillisecondsBehind {
				continue
			}

			require.Equal(t, metric.UnitMillisecondsMaximum, datum.Unit)
			require.Equal(t, metric.KindHistogram.Build(), datum.Kind)
			lagValues = append(lagValues, datum.Value)
		}
	}
	require.Equal(t, []float64{0, 9_000, 1_000}, lagValues)

	close(millisecondsBehind)
	<-done
}

type kinesisOtelMetricEmission struct {
	at   time.Time
	data metric.Data
}

type kinesisRecordsClient struct {
	Client
	output    *awsKinesis.GetRecordsOutput
	lastInput *awsKinesis.GetRecordsInput
	calls     int
}

func (c *kinesisRecordsClient) GetRecords(_ context.Context, input *awsKinesis.GetRecordsInput, _ ...func(*awsKinesis.Options)) (*awsKinesis.GetRecordsOutput, error) {
	c.calls++
	c.lastInput = input

	return c.output, nil
}

type kinesisOtelMetricWriter struct {
	clock             clock.Clock
	writer            metric.Writer
	mu                sync.Mutex
	recordedEmissions []kinesisOtelMetricEmission
	writeNotify       chan struct{}
}

func newKinesisOtelMetricWriter(t *testing.T, testClock clock.Clock) (*sdkmetric.ManualReader, *kinesisOtelMetricWriter) {
	t.Helper()

	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	t.Cleanup(func() {
		require.NoError(t, provider.Shutdown(t.Context()))
	})

	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))
	writer := metric.NewOtelWriterWithInterfaces(logger, provider.Meter("kinesis-emission-regression"))

	return reader, &kinesisOtelMetricWriter{
		clock:       testClock,
		writer:      writer,
		writeNotify: make(chan struct{}, 256),
	}
}

func (w *kinesisOtelMetricWriter) GetPriority() int {
	return w.writer.GetPriority()
}

func (w *kinesisOtelMetricWriter) Write(ctx context.Context, data metric.Data) {
	copiedData := make(metric.Data, len(data))
	for i, datum := range data {
		if datum == nil {
			continue
		}

		copiedDatum := *datum
		if datum.Dimensions != nil {
			copiedDatum.Dimensions = make(metric.Dimensions, len(datum.Dimensions))
			for key, value := range datum.Dimensions {
				copiedDatum.Dimensions[key] = value
			}
		}
		copiedData[i] = &copiedDatum
	}

	w.mu.Lock()
	w.recordedEmissions = append(w.recordedEmissions, kinesisOtelMetricEmission{
		at:   w.clock.Now(),
		data: copiedData,
	})
	w.mu.Unlock()

	w.writer.Write(ctx, data)
	w.writeNotify <- struct{}{}
}

func (w *kinesisOtelMetricWriter) WriteOne(ctx context.Context, datum *metric.Datum) {
	w.Write(ctx, metric.Data{datum})
}

func (w *kinesisOtelMetricWriter) emissions() []kinesisOtelMetricEmission {
	w.mu.Lock()
	defer w.mu.Unlock()

	return append([]kinesisOtelMetricEmission(nil), w.recordedEmissions...)
}

func waitForKinesisMetricWrites(t *testing.T, writer *kinesisOtelMetricWriter, count int) {
	t.Helper()

	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()

	for range count {
		select {
		case <-writer.writeNotify:
		case <-timer.C:
			t.Fatal("timed out waiting for Kinesis metric emission")
		}
	}
}

func collectKinesisOtelMetrics(t *testing.T, reader *sdkmetric.ManualReader) metricdata.ResourceMetrics {
	t.Helper()

	var exported metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(t.Context(), &exported))

	return exported
}

func kinesisOtelMetric(t *testing.T, exported metricdata.ResourceMetrics, name string) metricdata.Metrics {
	t.Helper()

	for _, scope := range exported.ScopeMetrics {
		for _, candidate := range scope.Metrics {
			if candidate.Name == name {
				return candidate
			}
		}
	}

	t.Fatalf("OpenTelemetry metric %q was not emitted", name)

	return metricdata.Metrics{}
}

func kinesisOtelStringAttribute(attributes attribute.Set, name string) (string, bool) {
	for _, keyValue := range attributes.ToSlice() {
		if string(keyValue.Key) == name {
			return keyValue.Value.AsString(), true
		}
	}

	return "", false
}

func kinesisOtelStringAttributeValue(t *testing.T, attributes attribute.Set, name string) string {
	t.Helper()

	value, ok := kinesisOtelStringAttribute(attributes, name)
	require.Truef(t, ok, "OpenTelemetry attribute %q was not emitted", name)

	return value
}

func assertKinesisBatchEmissions(t *testing.T, emissions []kinesisOtelMetricEmission, batchStart time.Time) {
	t.Helper()

	var totalCount float64
	var totalAt time.Time
	var failedAt []time.Time
	var reads int
	var processDurationSeen bool
	for _, emission := range emissions {
		for _, datum := range emission.data {
			switch datum.MetricName {
			case metricNameConsumedMessages:
				require.Equal(t, metric.UnitCount, datum.Unit)
				require.Equal(t, metric.KindCounter.Build(), datum.Kind)
				switch datum.Dimensions[metric.DimensionErrorType] {
				case metric.DimensionDefault:
					totalCount += datum.Value
					totalAt = emission.at
				case errorTypeProcessingFailed:
					require.Equal(t, float64(1), datum.Value)
					failedAt = append(failedAt, emission.at)
				default:
					t.Fatalf("unexpected consumed.messages error.type %q", datum.Dimensions[metric.DimensionErrorType])
				}
			case metricNameReadCount:
				require.Equal(t, float64(1), datum.Value)
				require.Equal(t, metric.UnitCount, datum.Unit)
				require.Equal(t, metric.KindCounter.Build(), datum.Kind)
				reads++
			case metricNameProcessDuration:
				require.Equal(t, float64(5*time.Minute.Milliseconds()), datum.Value)
				require.Equal(t, metric.UnitMillisecondsAverage, datum.Unit)
				require.Equal(t, metric.KindHistogram.Build(), datum.Kind)
				require.Equal(t, batchStart.Add(5*time.Minute), emission.at)
				processDurationSeen = true
			}
		}
	}
	require.Equal(t, float64(100), totalCount)
	require.Equal(t, batchStart.Add(5*time.Minute), totalAt)
	require.Len(t, failedAt, 10)
	for _, failureTime := range failedAt {
		require.Equal(t, batchStart.Add(2*time.Minute), failureTime)
	}
	require.Equal(t, 1, reads)
	require.True(t, processDurationSeen)
}
