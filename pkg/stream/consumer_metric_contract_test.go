package stream_test

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/log"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/metric"
	"github.com/justtrackio/gosoline/pkg/smpl"
	"github.com/justtrackio/gosoline/pkg/stream"
	"github.com/justtrackio/gosoline/pkg/tracing"
	"github.com/justtrackio/gosoline/pkg/uuid"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

const (
	metricProofWriterType = "stream_consumer_metric_proof"
	metricProofMeterName  = "stream.consumer.metric.proof"
	metricProofNameTotal  = "gosoline.stream.consumed.messages"
	metricProofNameRetry  = "gosoline.stream.retry.operations"
	metricProofNameTime   = "gosoline.stream.process.duration"
)

func TestConsumerMetricsMatchV2OtelContract(t *testing.T) {
	consumerNames := []string{
		"metric-proof-batch",
		"metric-proof-batch-retry",
		"metric-proof-single-retry",
		"metric-proof-ignored-model",
		"metric-proof-canceled",
	}
	writers := make(map[string]metric.Writer, len(consumerNames))
	for _, name := range consumerNames {
		writers[name] = newMetricProofWriter(name)
	}

	pipeline := startMetricProofPipeline(t)

	t.Run("batch totals include false and missing acknowledgments in the per-record failure subset", func(t *testing.T) {
		name := consumerNames[0]
		batches := make(chan []int, 2)
		callback := &metricProofBatchCallback{consume: func(_ context.Context, models []any, _ []map[string]string) ([]bool, error) {
			ids := metricProofModelIDs(models)
			batches <- ids

			if ids[0] == 4 {
				return []bool{true, false}, nil
			}

			return []bool{true, true, true}, nil
		}}
		primary := newMetricProofInput(metricProofMessages(t, 1, 2, 3, 4, 5, 6)...)
		retryInput := newMetricProofInput()
		output := stream.NewInMemoryOutput()
		consumer, err := newMetricProofBatchConsumer(t, name, writers[name], primary, retryInput, output, metricProofSettings(true), callback, stream.BatchConsumerSettings{BatchSize: 3, BufferSize: 6})
		require.NoError(t, err)
		cancel, done := startMetricProofConsumer(t, consumer)

		require.Equal(t, []int{1, 2, 3}, receiveMetricProofBatch(t, batches))
		require.Equal(t, []int{4, 5, 6}, receiveMetricProofBatch(t, batches))
		require.Eventually(t, func() bool { return output.Len() == 2 }, time.Second, 5*time.Millisecond)
		stopMetricProofConsumer(t, cancel, done)

		data := waitForMetricProofData(t, pipeline.reader, func(data metricdata.ResourceMetrics) bool {
			total, totalOK := metricCounterValue(data, metricProofNameTotal, name, nil, []string{"error.type"})
			failed, failedOK := metricCounterValue(data, metricProofNameTotal, name, map[string]string{"error.type": "processing_failed"}, nil)
			put, putOK := metricCounterValue(data, metricProofNameRetry, name, map[string]string{"operation": "put"}, nil)
			duration, durationOK := metricHistogramCount(data, metricProofNameTime, name)

			return totalOK && total == 6 && failedOK && failed == 2 && putOK && put == 2 && durationOK && duration == 2
		})

		assertMetricCounter(t, data, metricProofNameTotal, name, "{message}", map[string]string{"error.type": "processing_failed"}, nil, 2)
		assertMetricCounter(t, data, metricProofNameTotal, name, "{message}", nil, []string{"error.type"}, 6)
		assertMetricCounter(t, data, metricProofNameRetry, name, "{operation}", map[string]string{"operation": "put"}, nil, 2)
		assertMetricHistogram(t, data, name, 2)
		assertMetricProofInstrumentNames(t, data, name)
		require.Equal(t, 2, output.Len(), "each failed record is persisted independently")
	})

	t.Run("batch retry input records one get operation", func(t *testing.T) {
		name := consumerNames[1]
		batches := make(chan []int, 1)
		callback := &metricProofBatchCallback{consume: func(_ context.Context, models []any, _ []map[string]string) ([]bool, error) {
			batches <- metricProofModelIDs(models)

			return []bool{true}, nil
		}}
		primary := newMetricProofInput()
		retryMessage := metricProofMessage(t, 7)
		retryMessage.Attributes[stream.AttributeRetryId] = "batch-retry-7"
		retryInput := newMetricProofInput(retryMessage)
		output := stream.NewInMemoryOutput()
		consumer, err := newMetricProofBatchConsumer(t, name, writers[name], primary, retryInput, output, metricProofSettings(true), callback, stream.BatchConsumerSettings{BatchSize: 3, BufferSize: 3})
		require.NoError(t, err)
		cancel, done := startMetricProofConsumer(t, consumer)

		require.Equal(t, []int{7}, receiveMetricProofBatch(t, batches))
		require.True(t, receiveMetricProofAck(t, retryInput))
		stopMetricProofConsumer(t, cancel, done)

		data := waitForMetricProofData(t, pipeline.reader, func(data metricdata.ResourceMetrics) bool {
			get, getOK := metricCounterValue(data, metricProofNameRetry, name, map[string]string{"operation": "get"}, nil)
			total, totalOK := metricCounterValue(data, metricProofNameTotal, name, nil, []string{"error.type"})

			return getOK && get == 1 && totalOK && total == 1
		})
		assertMetricCounter(t, data, metricProofNameRetry, name, "{operation}", map[string]string{"operation": "get"}, nil, 1)
		assertMetricCounter(t, data, metricProofNameTotal, name, "{message}", nil, []string{"error.type"}, 1)
		assertMetricHistogram(t, data, name, 1)
	})

	t.Run("single retry failure records get, put, total, and one failure", func(t *testing.T) {
		name := consumerNames[2]
		callback := &metricProofSingleCallback{consume: func(context.Context, any, map[string]string) (bool, error) {
			return false, nil
		}}
		primary := newMetricProofInput()
		retryMessage := metricProofMessage(t, 8)
		retryMessage.Attributes[stream.AttributeRetryId] = "single-retry-8"
		retryInput := newMetricProofInput(retryMessage)
		output := stream.NewInMemoryOutput()
		consumer := newMetricProofSingleConsumer(t, name, writers[name], primary, retryInput, output, metricProofSettings(true), callback)
		cancel, done := startMetricProofConsumer(t, consumer)

		require.False(t, receiveMetricProofAck(t, retryInput))
		require.Eventually(t, func() bool { return output.Len() == 1 }, time.Second, 5*time.Millisecond)
		stopMetricProofConsumer(t, cancel, done)

		data := waitForMetricProofData(t, pipeline.reader, func(data metricdata.ResourceMetrics) bool {
			total, totalOK := metricCounterValue(data, metricProofNameTotal, name, nil, []string{"error.type"})
			failed, failedOK := metricCounterValue(data, metricProofNameTotal, name, map[string]string{"error.type": "processing_failed"}, nil)
			get, getOK := metricCounterValue(data, metricProofNameRetry, name, map[string]string{"operation": "get"}, nil)
			put, putOK := metricCounterValue(data, metricProofNameRetry, name, map[string]string{"operation": "put"}, nil)

			return totalOK && total == 1 && failedOK && failed == 1 && getOK && get == 1 && putOK && put == 1
		})
		assertMetricCounter(t, data, metricProofNameTotal, name, "{message}", nil, []string{"error.type"}, 1)
		assertMetricCounter(t, data, metricProofNameTotal, name, "{message}", map[string]string{"error.type": "processing_failed"}, nil, 1)
		assertMetricCounter(t, data, metricProofNameRetry, name, "{operation}", map[string]string{"operation": "get"}, nil, 1)
		assertMetricCounter(t, data, metricProofNameRetry, name, "{operation}", map[string]string{"operation": "put"}, nil, 1)
		assertMetricHistogram(t, data, name, 1)
	})

	t.Run("ignored model errors are counted as processed, not failed", func(t *testing.T) {
		name := consumerNames[3]
		consumed := make(chan struct{}, 1)
		callback := &metricProofSingleCallback{
			getModel: func(map[string]string) (any, error) { return nil, metricProofIgnorableModelError{} },
			consume: func(context.Context, any, map[string]string) (bool, error) {
				consumed <- struct{}{}

				return false, nil
			},
		}
		primary := newMetricProofInput(metricProofMessage(t, 9))
		retryInput := newMetricProofInput()
		settings := metricProofSettings(true)
		settings.IgnoreOnGetModelError.UnknownModel = true
		consumer := newMetricProofSingleConsumer(t, name, writers[name], primary, retryInput, stream.NewInMemoryOutput(), settings, callback)
		cancel, done := startMetricProofConsumer(t, consumer)
		require.True(t, receiveMetricProofAck(t, primary))
		stopMetricProofConsumer(t, cancel, done)
		select {
		case <-consumed:
			t.Fatal("ignored model reached Consume")
		default:
		}

		data := waitForMetricProofData(t, pipeline.reader, func(data metricdata.ResourceMetrics) bool {
			total, ok := metricCounterValue(data, metricProofNameTotal, name, nil, []string{"error.type"})

			return ok && total == 1
		})
		assertMetricCounter(t, data, metricProofNameTotal, name, "{message}", nil, []string{"error.type"}, 1)
		_, hasFailedSeries := metricCounterValue(data, metricProofNameTotal, name, map[string]string{"error.type": "processing_failed"}, nil)
		require.False(t, hasFailedSeries)
		assertMetricHistogram(t, data, name, 1)
		assertMetricProofInstrumentNames(t, data, name)
		require.Equal(t, 0, metricCountForConsumerName(data, name, "unknown_model"))
	})

	t.Run("canceled callback remains processed without a failed share", func(t *testing.T) {
		name := consumerNames[4]
		started := make(chan struct{}, 1)
		callback := &metricProofSingleCallback{consume: func(ctx context.Context, _ any, _ map[string]string) (bool, error) {
			started <- struct{}{}
			<-ctx.Done()

			return false, ctx.Err()
		}}
		primary := newMetricProofInput(metricProofMessage(t, 10))
		retryInput := newMetricProofInput()
		settings := metricProofSettings(false)
		settings.GraceTime = 10 * time.Millisecond
		consumer := newMetricProofSingleConsumer(t, name, writers[name], primary, retryInput, stream.NewInMemoryOutput(), settings, callback)
		cancel, done := startMetricProofConsumer(t, consumer)
		receiveMetricProofSignal(t, started)
		stopMetricProofConsumer(t, cancel, done)
		require.False(t, receiveMetricProofAck(t, primary))

		data := waitForMetricProofData(t, pipeline.reader, func(data metricdata.ResourceMetrics) bool {
			total, ok := metricCounterValue(data, metricProofNameTotal, name, nil, []string{"error.type"})

			return ok && total == 1
		})
		assertMetricCounter(t, data, metricProofNameTotal, name, "{message}", nil, []string{"error.type"}, 1)
		_, hasFailedSeries := metricCounterValue(data, metricProofNameTotal, name, map[string]string{"error.type": "processing_failed"}, nil)
		require.False(t, hasFailedSeries)
		assertMetricHistogram(t, data, name, 1)
		assertMetricProofInstrumentNames(t, data, name)
	})
}

type metricProofPipeline struct {
	reader   *sdkmetric.ManualReader
	provider *sdkmetric.MeterProvider
	cancel   context.CancelFunc
	done     chan error
}

func startMetricProofPipeline(t *testing.T) *metricProofPipeline {
	t.Helper()

	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))
	metric.RegisterWriterFactory(metricProofWriterType, func(_ context.Context, _ cfg.Config, factoryLogger log.Logger) (metric.Writer, error) {
		return metric.NewOtelWriterWithInterfaces(factoryLogger, provider.Meter(metricProofMeterName)), nil
	})

	config := cfg.New(map[string]any{
		"metric": map[string]any{
			"enabled":  true,
			"interval": "1h",
			"writers":  []string{metricProofWriterType},
		},
	})
	ctx := appctx.WithContainer(t.Context())
	module, err := metric.NewDaemonModule(ctx, config, logger)
	require.NoError(t, err)
	require.NotNil(t, module)

	daemonCtx, cancel := context.WithCancel(ctx)
	done := make(chan error, 1)
	go func() {
		done <- module.Run(daemonCtx)
		close(done)
	}()

	pipeline := &metricProofPipeline{reader: reader, provider: provider, cancel: cancel, done: done}
	t.Cleanup(func() { pipeline.stop(t) })

	return pipeline
}

func (p *metricProofPipeline) stop(t *testing.T) {
	t.Helper()
	p.cancel()
	select {
	case err := <-p.done:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("metric daemon did not stop")
	}
	require.NoError(t, p.provider.Shutdown(context.Background()))
}

func newMetricProofWriter(consumer string) metric.Writer {
	return metric.NewWriter("stream",
		&metric.Datum{
			Priority:   metric.PriorityHigh,
			MetricName: "consumed.messages",
			Dimensions: map[string]string{
				"consumer.name":           consumer,
				metric.DimensionErrorType: metric.DimensionDefault,
			},
			Unit:  metric.UnitCount,
			Value: 0,
		},
		metricProofRetryDefault(consumer, "get"),
		metricProofRetryDefault(consumer, "put"),
	)
}

func metricProofRetryDefault(consumer, operation string) *metric.Datum {
	return &metric.Datum{
		Priority:   metric.PriorityHigh,
		MetricName: "retry.operations",
		Dimensions: map[string]string{"consumer.name": consumer, "operation": operation},
		Unit:       metric.UnitCount,
		Value:      0,
		Kind:       metric.KindCounter.Build(),
	}
}

func newMetricProofSingleConsumer(t *testing.T, name string, writer metric.Writer, input, retryInput stream.Input, output *stream.InMemoryOutput, settings stream.ConsumerSettings, callback stream.UntypedConsumerCallback) *stream.Consumer {
	base := stream.NewConsumerBaseWithInterfaces(
		uuid.New(),
		logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t)),
		writer,
		tracing.NewLocalTracer(),
		input,
		stream.NewMessageEncoder(&stream.MessageEncoderSettings{Encoding: stream.EncodingJson}),
		retryInput,
		stream.NewRetryHandlerSqsWithInterfaces(output, &stream.RetryHandlerSqsSettings{
			RetryHandlerSettings: stream.RetryHandlerSettings{After: time.Second, MaxAttempts: 3},
		}),
		settings,
		name,
		metricProofSamplingDecider{},
		clock.Provider,
	)

	return stream.NewUntypedConsumerWithInterfaces(base, callback)
}

func newMetricProofBatchConsumer(t *testing.T, name string, writer metric.Writer, input, retryInput stream.Input, output *stream.InMemoryOutput, settings stream.ConsumerSettings, callback stream.UntypedBatchConsumerCallback, batchSettings stream.BatchConsumerSettings) (*stream.BatchConsumer, error) {
	base := stream.NewConsumerBaseWithInterfaces(
		uuid.New(),
		logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t)),
		writer,
		tracing.NewLocalTracer(),
		input,
		stream.NewMessageEncoder(&stream.MessageEncoderSettings{Encoding: stream.EncodingJson}),
		retryInput,
		stream.NewRetryHandlerSqsWithInterfaces(output, &stream.RetryHandlerSqsSettings{
			RetryHandlerSettings: stream.RetryHandlerSettings{After: time.Second, MaxAttempts: 3},
		}),
		settings,
		name,
		metricProofSamplingDecider{},
		clock.Provider,
	)

	return stream.NewUntypedBatchConsumerWithInterfaces(base, callback, batchSettings)
}

func metricProofSettings(retryEnabled bool) stream.ConsumerSettings {
	return stream.ConsumerSettings{
		Input:                "metric-proof",
		Encoding:             stream.EncodingJson,
		IdleTimeout:          time.Second,
		GraceTime:            time.Second,
		Retry:                stream.ConsumerRetrySettings{Enabled: retryEnabled, GraceTime: time.Second},
		AggregateMessageMode: stream.AggregateMessageModeAtMostOnce,
	}
}

type metricProofSamplingDecider struct{}

func (metricProofSamplingDecider) Decide(ctx context.Context, _ ...smpl.Strategy) (context.Context, bool, error) {
	return ctx, false, nil
}

type metricProofModule interface {
	Run(context.Context) error
}

func startMetricProofConsumer(t *testing.T, consumer metricProofModule) (cancel context.CancelFunc, done chan error) {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	done = make(chan error, 1)
	go func() {
		done <- consumer.Run(ctx)
		close(done)
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Error("stream consumer did not stop")
		}
	})

	return cancel, done
}

func stopMetricProofConsumer(t *testing.T, cancel context.CancelFunc, done <-chan error) {
	t.Helper()
	cancel()
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("stream consumer did not stop")
	}
}

type metricProofInput struct {
	messages  []*stream.Message
	processed chan bool
	stopped   chan struct{}
	stopOnce  sync.Once
}

func newMetricProofInput(messages ...*stream.Message) *metricProofInput {
	return &metricProofInput{messages: messages, processed: make(chan bool, len(messages)), stopped: make(chan struct{})}
}

func (i *metricProofInput) Run(ctx context.Context, process stream.InputProcess) error {
	for _, message := range i.messages {
		ack := process(ctx, message)
		i.processed <- ack
	}

	select {
	case <-ctx.Done():
	case <-i.stopped:
	}

	return nil
}

func (i *metricProofInput) Stop(context.Context) {
	i.stopOnce.Do(func() { close(i.stopped) })
}

func (*metricProofInput) IsHealthy() bool { return true }

type metricProofModel struct {
	ID int `json:"id"`
}

type metricProofSingleCallback struct {
	getModel func(map[string]string) (any, error)
	consume  func(context.Context, any, map[string]string) (bool, error)
}

func (c *metricProofSingleCallback) GetModel(attributes map[string]string) (any, error) {
	if c.getModel != nil {
		return c.getModel(attributes)
	}

	return new(metricProofModel), nil
}

func (c *metricProofSingleCallback) Consume(ctx context.Context, model any, attributes map[string]string) (bool, error) {
	return c.consume(ctx, model, attributes)
}

type metricProofBatchCallback struct {
	consume func(context.Context, []any, []map[string]string) ([]bool, error)
}

func (*metricProofBatchCallback) GetModel(map[string]string) (any, error) {
	return new(metricProofModel), nil
}

func (c *metricProofBatchCallback) Consume(ctx context.Context, models []any, attributes []map[string]string) ([]bool, error) {
	return c.consume(ctx, models, attributes)
}

type metricProofIgnorableModelError struct{}

func (metricProofIgnorableModelError) Error() string { return "unknown model" }
func (metricProofIgnorableModelError) IsIgnorableWithSettings(settings stream.IgnoreOnGetModelErrorSettings) bool {
	return settings.UnknownModel
}

func metricProofMessages(t *testing.T, ids ...int) []*stream.Message {
	t.Helper()
	messages := make([]*stream.Message, 0, len(ids))
	for _, id := range ids {
		messages = append(messages, metricProofMessage(t, id))
	}

	return messages
}

func metricProofMessage(t *testing.T, id int) *stream.Message {
	t.Helper()
	message, err := stream.MarshalJsonMessage(metricProofModel{ID: id})
	require.NoError(t, err)

	return message
}

func metricProofModelIDs(models []any) []int {
	ids := make([]int, 0, len(models))
	for _, model := range models {
		ids = append(ids, model.(*metricProofModel).ID)
	}

	return ids
}

func receiveMetricProofBatch(t *testing.T, batches <-chan []int) []int {
	t.Helper()
	select {
	case batch := <-batches:
		return batch
	case <-time.After(2 * time.Second):
		t.Fatal("batch callback did not run")

		return nil
	}
}

func receiveMetricProofAck(t *testing.T, input *metricProofInput) bool {
	t.Helper()
	select {
	case ack := <-input.processed:
		return ack
	case <-time.After(2 * time.Second):
		t.Fatal("input message was not processed")

		return false
	}
}

func receiveMetricProofSignal(t *testing.T, signal <-chan struct{}) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(2 * time.Second):
		t.Fatal("callback did not start")
	}
}

func waitForMetricProofData(t *testing.T, reader *sdkmetric.ManualReader, ready func(metricdata.ResourceMetrics) bool) metricdata.ResourceMetrics {
	t.Helper()
	var data metricdata.ResourceMetrics
	var collectErr error
	require.Eventually(t, func() bool {
		data = metricdata.ResourceMetrics{}
		collectErr = reader.Collect(t.Context(), &data)

		return collectErr == nil && ready(data)
	}, 2*time.Second, 5*time.Millisecond)
	require.NoError(t, collectErr)

	return data
}

func findMetric(data metricdata.ResourceMetrics, name string) (metricdata.Metrics, bool) {
	for _, scope := range data.ScopeMetrics {
		for _, metric := range scope.Metrics {
			if metric.Name == name {
				return metric, true
			}
		}
	}

	return metricdata.Metrics{}, false
}

func metricCounterValue(data metricdata.ResourceMetrics, name, consumer string, dimensions map[string]string, absent []string) (float64, bool) {
	instrument, ok := findMetric(data, name)
	if !ok {
		return 0, false
	}
	counter, ok := instrument.Data.(metricdata.Sum[float64])
	if !ok || !counter.IsMonotonic {
		return 0, false
	}

	var value float64
	var found bool
	for _, point := range counter.DataPoints {
		if metricAttribute(point.Attributes, "consumer.name") != consumer || !metricAttributesMatch(point.Attributes, dimensions) {
			continue
		}
		if metricHasAnyAttribute(point.Attributes, absent) {
			continue
		}
		value += point.Value
		found = true
	}

	return value, found
}

func metricHistogramCount(data metricdata.ResourceMetrics, name, consumer string) (uint64, bool) {
	instrument, ok := findMetric(data, name)
	if !ok {
		return 0, false
	}
	histogram, ok := instrument.Data.(metricdata.Histogram[float64])
	if !ok {
		return 0, false
	}

	var count uint64
	var found bool
	for _, point := range histogram.DataPoints {
		if metricAttribute(point.Attributes, "consumer.name") == consumer {
			count += point.Count
			found = true
		}
	}

	return count, found
}

func assertMetricCounter(t *testing.T, data metricdata.ResourceMetrics, name, consumer, unit string, dimensions map[string]string, absent []string, want float64) {
	t.Helper()
	instrument, ok := findMetric(data, name)
	require.True(t, ok, "missing OTel instrument %s", name)
	require.Equal(t, unit, instrument.Unit)
	counter, ok := instrument.Data.(metricdata.Sum[float64])
	require.True(t, ok, "%s must export as a counter", name)
	require.True(t, counter.IsMonotonic, "%s must be monotonic", name)
	value, found := metricCounterValue(data, name, consumer, dimensions, absent)
	require.True(t, found, "missing %s series for consumer %s", name, consumer)
	require.Equal(t, want, value)
}

func assertMetricHistogram(t *testing.T, data metricdata.ResourceMetrics, consumer string, wantCount uint64) {
	t.Helper()
	instrument, ok := findMetric(data, metricProofNameTime)
	require.True(t, ok, "missing OTel instrument %s", metricProofNameTime)
	require.Equal(t, "s", instrument.Unit)
	histogram, ok := instrument.Data.(metricdata.Histogram[float64])
	require.True(t, ok, "processing duration must export as a histogram")

	var count uint64
	var found bool
	for _, point := range histogram.DataPoints {
		if metricAttribute(point.Attributes, "consumer.name") == consumer {
			count += point.Count
			found = true
		}
	}
	require.True(t, found, "missing duration histogram point for consumer %s", consumer)
	require.Equal(t, wantCount, count)
}

func assertMetricProofInstrumentNames(t *testing.T, data metricdata.ResourceMetrics, consumer string) {
	t.Helper()
	got := metricInstrumentNamesForConsumer(data, consumer)
	require.ElementsMatch(t, []string{metricProofNameTotal, metricProofNameRetry, metricProofNameTime}, got)
}

func metricCountForConsumerName(data metricdata.ResourceMetrics, consumer, namePart string) int {
	count := 0
	for _, scope := range data.ScopeMetrics {
		for _, instrument := range scope.Metrics {
			if !strings.Contains(instrument.Name, namePart) {
				continue
			}
			forEachMetricProofAttributes(instrument.Data, func(attributes attribute.Set) {
				if metricAttribute(attributes, "consumer.name") == consumer {
					count++
				}
			})
		}
	}

	return count
}

func metricInstrumentNamesForConsumer(data metricdata.ResourceMetrics, consumer string) []string {
	var names []string
	for _, scope := range data.ScopeMetrics {
		for _, instrument := range scope.Metrics {
			matched := false
			forEachMetricProofAttributes(instrument.Data, func(attributes attribute.Set) {
				matched = matched || metricAttribute(attributes, "consumer.name") == consumer
			})
			if matched {
				names = append(names, instrument.Name)
			}
		}
	}

	return names
}

func forEachMetricProofAttributes(data any, visit func(attribute.Set)) {
	switch points := data.(type) {
	case metricdata.Sum[float64]:
		for _, point := range points.DataPoints {
			visit(point.Attributes)
		}
	case metricdata.Histogram[float64]:
		for _, point := range points.DataPoints {
			visit(point.Attributes)
		}
	case metricdata.Gauge[float64]:
		for _, point := range points.DataPoints {
			visit(point.Attributes)
		}
	}
}

func metricAttribute(attributes attribute.Set, key string) string {
	for _, item := range attributes.ToSlice() {
		if string(item.Key) == key {
			return item.Value.AsString()
		}
	}

	return ""
}

func metricAttributesMatch(attributes attribute.Set, expected map[string]string) bool {
	for key, value := range expected {
		if metricAttribute(attributes, key) != value {
			return false
		}
	}

	return true
}

func metricHasAnyAttribute(attributes attribute.Set, keys []string) bool {
	for _, key := range keys {
		for _, item := range attributes.ToSlice() {
			if string(item.Key) == key {
				return true
			}
		}
	}

	return false
}
