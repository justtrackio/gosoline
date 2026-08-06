package stream_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/mdl"
	metricMocks "github.com/justtrackio/gosoline/pkg/metric/mocks"
	"github.com/justtrackio/gosoline/pkg/smpl"
	smplMocks "github.com/justtrackio/gosoline/pkg/smpl/mocks"
	"github.com/justtrackio/gosoline/pkg/stream"
	"github.com/justtrackio/gosoline/pkg/stream/health"
	"github.com/justtrackio/gosoline/pkg/stream/mocks"
	"github.com/justtrackio/gosoline/pkg/test/matcher"
	"github.com/justtrackio/gosoline/pkg/tracing"
	uuidMocks "github.com/justtrackio/gosoline/pkg/uuid/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kgo"
)

// shutdownConsumerFixture wires a Consumer around an arbitrary input so we can observe what happens to the messages
// which are still in flight when the kernel context gets cancelled.
type shutdownConsumerFixture struct {
	consumer     *stream.Consumer
	callback     *mocks.RunnableUntypedConsumerCallback
	kernelCtx    context.Context
	kernelCancel context.CancelFunc
	inputData    chan *stream.Message
}

func newShutdownConsumerFixture(t *testing.T, input stream.Input, inputData chan *stream.Message, retryEnabled bool) *shutdownConsumerFixture {
	t.Helper()

	kernelCtx, kernelCancel := context.WithCancel(t.Context())

	callback := mocks.NewRunnableUntypedConsumerCallback(t)
	callback.EXPECT().Run(matcher.Context).Return(nil).Once()

	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))
	mw := metricMocks.NewWriter(t)
	mw.EXPECT().Write(matcher.Context, mock.Anything).Return().Maybe()

	settings := stream.ConsumerSettings{
		Input:            "test",
		RunnerCount:      1,
		IdleTimeout:      time.Second,
		ConsumeGraceTime: 20 * time.Millisecond,
		Retry: stream.ConsumerRetrySettings{
			Enabled: retryEnabled,
		},
		Healthcheck: health.HealthCheckSettings{
			Timeout: time.Minute,
		},
		AggregateMessageMode: stream.AggregateMessageModeAtMostOnce,
	}

	samplingDecider := smplMocks.NewDecider(t)
	samplingDecider.EXPECT().Decide(matcher.Context).RunAndReturn(func(ctx context.Context, strategy ...smpl.Strategy) (context.Context, bool, error) {
		return ctx, false, nil
	}).Maybe()

	retryInput := stream.NewNoopInput()

	baseConsumer := stream.NewBaseConsumerWithInterfaces(
		uuidMocks.NewUuid(t),
		logger,
		mw,
		tracing.NewLocalTracer(),
		input,
		stream.NewMessageEncoder(&stream.MessageEncoderSettings{}),
		retryInput,
		stream.NewRetryHandlerNoopWithInterfaces(),
		callback,
		settings,
		"test",
		cfg.Identity{},
	)

	return &shutdownConsumerFixture{
		consumer:     stream.NewUntypedConsumerWithInterfaces(baseConsumer, callback, clock.NewHealthCheckTimerWithInterfaces(clock.NewFakeClock(), settings.Healthcheck.Timeout), samplingDecider),
		callback:     callback,
		kernelCtx:    kernelCtx,
		kernelCancel: kernelCancel,
		inputData:    inputData,
	}
}

// TestConsumerShutdown_DrainsInputWhichCanNotRedeliver makes sure we attempt to finish messages which are already in
// flight once the kernel context got cancelled. For Kafka, dropping them would complete and commit the containing batch
// without processing them.
func TestConsumerShutdown_DrainsInputWhichCanNotRedeliver(t *testing.T) {
	inputData := make(chan *stream.Message, 10)
	var stopOnce sync.Once

	// Kafka is acknowledgeable so it can track completion, but acknowledgements cannot redeliver individual records.
	input := mocks.NewAcknowledgeableInput(t)
	input.EXPECT().Data().Return(inputData)
	input.EXPECT().Stop(matcher.Context).Run(func(context.Context) {
		stopOnce.Do(func() { close(inputData) })
	}).Once()
	input.EXPECT().Ack(matcher.Context, mock.Anything, true).Return(nil).Twice()

	fixture := newShutdownConsumerFixture(t, input, inputData, false)

	input.EXPECT().
		Run(matcher.Context).
		Run(func(ctx context.Context) {
			// cancel the kernel first, so every message below is consumed during shutdown
			fixture.kernelCancel()
			inputData <- stream.NewJsonMessage(`"foo"`)
			inputData <- stream.NewJsonMessage(`"bar"`)
		}).
		Return(nil).
		Once()

	var consumed atomic.Int32
	fixture.callback.EXPECT().GetModel(mock.Anything).Return(mdl.Box(""), nil).Twice()
	fixture.callback.EXPECT().
		Consume(matcher.Context, mock.Anything, mock.Anything).
		Run(func(context.Context, any, map[string]string) {
			consumed.Add(1)
		}).
		Return(true, nil).
		Twice()

	err := fixture.consumer.Run(fixture.kernelCtx)
	require.NoError(t, err)

	require.Equal(t, int32(2), consumed.Load(), "in flight messages must still be processed during shutdown")
}

type redeliveringInput struct {
	stream.AcknowledgeableInput
}

func (i redeliveringInput) GetRetryHandler() (stream.Input, stream.RetryHandler) {
	return stream.NewNoopInput(), stream.NewRetryHandlerNoopWithInterfaces()
}

// TestConsumerShutdown_DrainsInputWhichCanRedeliver applies the same shutdown behavior to an input which could
// redeliver. This keeps shutdown independent of transport capabilities and gives all in-flight work the configured
// consume grace time.
func TestConsumerShutdown_DrainsInputWhichCanRedeliver(t *testing.T) {
	inputData := make(chan *stream.Message, 10)
	var stopOnce sync.Once

	input := mocks.NewAcknowledgeableInput(t)
	input.EXPECT().Data().Return(inputData)
	input.EXPECT().Stop(matcher.Context).Run(func(context.Context) {
		stopOnce.Do(func() { close(inputData) })
	}).Once()
	input.EXPECT().Ack(matcher.Context, mock.Anything, true).Return(nil).Twice()

	fixture := newShutdownConsumerFixture(t, redeliveringInput{AcknowledgeableInput: input}, inputData, false)

	input.EXPECT().
		Run(matcher.Context).
		Run(func(ctx context.Context) {
			fixture.kernelCancel()
			inputData <- stream.NewJsonMessage(`"foo"`)
			inputData <- stream.NewJsonMessage(`"bar"`)
		}).
		Return(nil).
		Once()

	var consumed atomic.Int32
	fixture.callback.EXPECT().GetModel(mock.Anything).Return(mdl.Box(""), nil).Twice()
	fixture.callback.EXPECT().
		Consume(matcher.Context, mock.Anything, mock.Anything).
		Run(func(context.Context, any, map[string]string) {
			consumed.Add(1)
		}).
		Return(true, nil).
		Twice()

	err := fixture.consumer.Run(fixture.kernelCtx)
	require.NoError(t, err)
	require.Equal(t, int32(2), consumed.Load(), "in-flight messages should be processed regardless of redelivery support")
}

func TestConsumerShutdown_CancelsProcessingAfterGraceTime(t *testing.T) {
	inputData := make(chan *stream.Message, 1)
	var stopOnce sync.Once

	input := mocks.NewAcknowledgeableInput(t)
	input.EXPECT().Data().Return(inputData)
	input.EXPECT().Stop(matcher.Context).Run(func(context.Context) {
		stopOnce.Do(func() { close(inputData) })
	}).Once()
	input.EXPECT().Ack(matcher.Context, mock.Anything, false).Return(nil).Once()

	fixture := newShutdownConsumerFixture(t, input, inputData, false)
	input.EXPECT().
		Run(matcher.Context).
		Run(func(context.Context) {
			fixture.kernelCancel()
			inputData <- stream.NewJsonMessage(`"foo"`)
		}).
		Return(nil).
		Once()

	callbackCancelled := make(chan struct{})
	fixture.callback.EXPECT().GetModel(mock.Anything).Return(mdl.Box(""), nil).Once()
	fixture.callback.EXPECT().
		Consume(matcher.Context, mock.Anything, mock.Anything).
		RunAndReturn(func(ctx context.Context, _ any, _ map[string]string) (bool, error) {
			<-ctx.Done()
			close(callbackCancelled)

			return false, ctx.Err()
		}).
		Once()

	err := fixture.consumer.Run(fixture.kernelCtx)
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		select {
		case <-callbackCancelled:
			return true
		default:
			return false
		}
	}, time.Second, time.Millisecond, "the callback context should be cancelled after the consume grace time")
}

// TestKafkaMessageHandlerCompletion covers the handshake the Kafka partition consumer relies on: the batch is only
// reported as done once every single record of it was acknowledged.
func TestKafkaMessageHandlerCompletion(t *testing.T) {
	data := make(chan *stream.Message, 4)
	handler := stream.NewKafkaMessageHandler(data)

	records := []*kgo.Record{
		{Topic: "topic", Partition: 1, Offset: 1, Value: []byte(`"foo"`)},
		nil, // nil records are skipped and must not be counted
		{Topic: "topic", Partition: 1, Offset: 2, Value: []byte(`"bar"`)},
	}

	completion := handler.Handle(records)

	select {
	case <-completion.Done():
		t.Fatal("the batch must not be done before its records were acknowledged")
	default:
	}

	input := stream.NewKafkaInputWithInterfaces(nil, nil, data)
	ackInput, ok := input.(stream.AcknowledgeableInput)
	require.True(t, ok, "the kafka input has to be acknowledgeable")

	msg1, msg2 := <-data, <-data

	require.NoError(t, ackInput.Ack(t.Context(), msg1, true))
	select {
	case <-completion.Done():
		t.Fatal("the batch must not be done while one of its records is still being processed")
	default:
	}

	require.NoError(t, ackInput.Ack(t.Context(), msg2, false))
	select {
	case <-completion.Done():
	case <-time.After(time.Second):
		t.Fatal("the batch should be done once all of its records were acknowledged")
	}

	require.Equal(t, 1, completion.FailedCount())
}
