package stream_test

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/log"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	metricMocks "github.com/justtrackio/gosoline/pkg/metric/mocks"
	"github.com/justtrackio/gosoline/pkg/stream"
	"github.com/justtrackio/gosoline/pkg/stream/health"
	streamMocks "github.com/justtrackio/gosoline/pkg/stream/mocks"
	"github.com/justtrackio/gosoline/pkg/test/matcher"
	"github.com/justtrackio/gosoline/pkg/tracing"
	"github.com/justtrackio/gosoline/pkg/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// TestBatchConsumerTestSuite runs the batch admission, processing, retry and lifecycle scenarios.
func TestBatchConsumerTestSuite(t *testing.T) {
	suite.Run(t, new(BatchConsumerTestSuite))
}

// BatchConsumerTestSuite exercises batching, admission, retries and lifecycle with independent fixtures.
type BatchConsumerTestSuite struct {
	suite.Suite

	consumer      *stream.BatchConsumer
	input         *batchTestInput
	cancel        context.CancelFunc
	done          chan struct{}
	err           error
	retryInput    *batchTestRetryInput
	retryMessages chan *stream.Message
	retryWriteErr error
	output        *streamMocks.Output
	logger        log.Logger
	metrics       *metricMocks.Writer
	settings      stream.ConsumerSettings
	batchSettings stream.BatchConsumerSettings
}

// Each table-driven case gets independent inputs, channels and mock expectations.
func (s *BatchConsumerTestSuite) SetupSubTest() {
	s.SetupTest()
}

func (s *BatchConsumerTestSuite) SetupTest() {
	t := s.T()
	s.consumer, s.cancel, s.err, s.retryWriteErr = nil, nil, nil, nil
	s.done = make(chan struct{})
	s.input = &batchTestInput{InMemoryInput: stream.NewInMemoryInput(&stream.InMemorySettings{Size: 20, RunnerCount: 1}), admitted: make(chan bool, 100)}
	s.retryInput = &batchTestRetryInput{InMemoryInput: stream.NewInMemoryInput(&stream.InMemorySettings{Size: 20, RunnerCount: 1}), results: make(chan bool, 100)}
	s.retryMessages = make(chan *stream.Message, 100)
	s.output = streamMocks.NewOutput(t)
	s.output.EXPECT().WriteOne(matcher.Context, mock.Anything).RunAndReturn(func(_ context.Context, message stream.WritableMessage) error {
		if s.retryWriteErr != nil {
			return s.retryWriteErr
		}

		msg := message.(*stream.Message)
		s.retryMessages <- msg
		s.retryInput.Publish(msg)

		return nil
	}).Maybe()
	s.logger = logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))
	s.metrics = metricMocks.NewWriter(t)
	s.metrics.EXPECT().Write(matcher.Context, mock.Anything).Maybe()
	s.settings = stream.ConsumerSettings{
		Input: "consumer", IdleTimeout: time.Hour, GraceTime: 100 * time.Millisecond,
		Retry:       stream.ConsumerRetrySettings{Enabled: true, Type: "sqs", GraceTime: 10 * time.Millisecond},
		Healthcheck: health.HealthCheckSettings{Timeout: 10 * time.Millisecond},
	}
	s.batchSettings = stream.BatchConsumerSettings{BatchSize: 1}
}

// TestAcknowledgesAdmissionAndFlushesOnSize verifies that primary messages are acknowledged
// before business processing completes, and a partial batch waits until the size threshold is met.
func (s *BatchConsumerTestSuite) TestAcknowledgesAdmissionAndFlushesOnSize() {
	t := s.T()
	started, release := make(chan struct{}), make(chan struct{})
	defer close(release)
	var got []any
	s.startConsumer(2, time.Hour, func(ctx context.Context, models []any, attributes []map[string]string) ([]bool, error) {
		got = models
		close(started)
		<-release

		return successfulBatch(ctx, models, attributes)
	})
	s.publishModel(1)

	select {
	case <-started:
		t.Fatal("partial batch processed before size or time trigger")
	default:
	}

	s.publishModel(2)
	waitBatchTest(t, started)
	require.Equal(t, []any{&batchTestModel{Value: 1}, &batchTestModel{Value: 2}}, got)
	release <- struct{}{}
	s.cancel()
	waitBatchTest(t, s.done)
	require.NoError(t, s.err)
}

// TestFlushesOnTimerAndOnInputCompletion verifies that a batch below the size threshold
// flushes either when its timer fires or when the inputs finish.
func (s *BatchConsumerTestSuite) TestFlushesOnTimerAndOnInputCompletion() {
	for _, interval := range []time.Duration{5 * time.Millisecond, time.Hour} {
		s.Run(interval.String(), func() {
			t := s.T()
			processed := make(chan struct{})
			s.startConsumer(100, interval, func(ctx context.Context, models []any, attributes []map[string]string) ([]bool, error) {
				close(processed)

				return successfulBatch(ctx, models, attributes)
			})
			s.publishModel(1)

			if interval == time.Hour {
				s.input.Stop(t.Context())
			}

			waitBatchTest(t, processed)
			s.cancel()
			waitBatchTest(t, s.done)
			require.NoError(t, s.err)
		})
	}
}

// TestRetriesOnlyFailedRecords verifies that a partially successful batch retries only
// the failed record, even when the callback also returns an error.
func (s *BatchConsumerTestSuite) TestRetriesOnlyFailedRecords() {
	t := s.T()
	var calls atomic.Int32
	retried := make(chan int, 1)
	s.startConsumer(2, 5*time.Millisecond, func(_ context.Context, models []any, _ []map[string]string) ([]bool, error) {
		if calls.Add(1) == 1 {
			return []bool{true, false}, errors.New("temporary failure")
		}

		retried <- models[0].(*batchTestModel).Value

		return []bool{true}, nil
	})
	s.publishModel(1)
	s.publishModel(2)

	select {
	case value := <-retried:
		require.Equal(t, 2, value)
	case <-time.After(time.Second):
		t.Fatal("failed record was not retried")
	}

	s.cancel()
	waitBatchTest(t, s.done)
	require.NoError(t, s.err)
	require.EqualValues(t, 2, calls.Load())
}

// TestUsesRetryInputRedeliveryAndRecoversPanics covers callback errors and panics:
// primary work is queued once, and failed retries remain unacknowledged for three simulated receives.
func (s *BatchConsumerTestSuite) TestUsesRetryInputRedeliveryAndRecoversPanics() {
	for _, panicCallback := range []bool{false, true} {
		s.Run(fmt.Sprint(panicCallback), func() {
			t := s.T()
			var calls atomic.Int32
			s.startConsumer(1, time.Millisecond, func(context.Context, []any, []map[string]string) ([]bool, error) {
				calls.Add(1)
				if panicCallback {
					panic("failed")
				}

				return nil, errors.New("failed")
			})
			s.publishModel(1)

			for range 3 {
				select {
				case ack := <-s.retryInput.results:
					require.False(t, ack)
				case <-time.After(time.Second):
					t.Fatal("retry input did not receive failure acknowledgement")
				}
			}

			s.cancel()
			waitBatchTest(t, s.done)
			require.NoError(t, s.err)
			require.EqualValues(t, 4, calls.Load())
			require.Len(t, s.retryMessages, 1)
		})
	}
}

// TestHealthTracksBusinessProcessing verifies that a blocked business callback makes
// the consumer unhealthy after its processing timeout, despite already acknowledging admission.
func (s *BatchConsumerTestSuite) TestHealthTracksBusinessProcessing() {
	t := s.T()
	started, release := make(chan struct{}), make(chan struct{})
	defer close(release)
	s.startConsumer(1, time.Hour, func(ctx context.Context, models []any, attributes []map[string]string) ([]bool, error) {
		close(started)
		<-release

		return successfulBatch(ctx, models, attributes)
	})
	s.publishModel(1)
	waitBatchTest(t, started)
	require.Eventually(t, func() bool {
		healthy, err := s.consumer.IsHealthy(t.Context())

		return err == nil && !healthy
	}, time.Second, time.Millisecond)
	release <- struct{}{}
	s.cancel()
	waitBatchTest(t, s.done)
	require.NoError(t, s.err)
}

// TestSharedShutdownDeadlineCancelsWork verifies that shutdown cancels a blocked callback
// when the shared drain deadline expires and persists its unsuccessful primary record for retry.
func (s *BatchConsumerTestSuite) TestSharedShutdownDeadlineCancelsWork() {
	t := s.T()
	started := make(chan struct{})
	s.startConsumer(1, time.Hour, func(ctx context.Context, _ []any, _ []map[string]string) ([]bool, error) {
		close(started)
		<-ctx.Done()

		return []bool{false}, ctx.Err()
	})
	s.publishModel(1)
	waitBatchTest(t, started)
	s.cancel()
	waitBatchTest(t, s.done)
	require.NoError(t, s.err)
	require.Len(t, s.retryMessages, 1)
}

// TestBackpressureDoesNotRequireConcurrentInputRunners verifies that a single input runner
// blocks admission when the batch channel is full and resumes without losing records once processing advances.
func (s *BatchConsumerTestSuite) TestBackpressureDoesNotRequireConcurrentInputRunners() {
	t := s.T()
	started, release := make(chan struct{}), make(chan struct{})
	releaseBatch := func() {
		select {
		case <-release:
		default:
			close(release)
		}
	}
	defer releaseBatch()
	var calls atomic.Int32
	s.startConsumer(1, time.Hour, func(ctx context.Context, models []any, attributes []map[string]string) ([]bool, error) {
		if calls.Add(1) == 1 {
			close(started)
		}

		<-release

		return successfulBatch(ctx, models, attributes)
	})
	s.publishModel(1)
	waitBatchTest(t, started)
	s.publishModel(2)
	third, err := stream.MarshalJsonMessage(batchTestModel{Value: 3})
	require.NoError(t, err)
	s.input.Publish(third)

	select {
	case <-s.input.admitted:
		t.Fatal("admitted more work while the bounded channel was full")
	case <-time.After(10 * time.Millisecond):
	}

	releaseBatch()
	select {
	case ack := <-s.input.admitted:
		require.True(t, ack)
	case <-time.After(time.Second):
		t.Fatal("admission did not resume after processing")
	}

	s.cancel()
	waitBatchTest(t, s.done)
	require.NoError(t, s.err)
	require.EqualValues(t, 3, calls.Load())
}

// TestRunnableFailureStopsInputs verifies that an optional background callback's error
// stops the consumer and is returned by Run.
func (s *BatchConsumerTestSuite) TestRunnableFailureStopsInputs() {
	t := s.T()
	expected := errors.New("runnable failed")
	s.startConsumerWithCallback(2, time.Hour, &batchTestCallback{
		consume: successfulBatch,
		run:     func(context.Context) error { return expected },
	})
	waitBatchTest(t, s.done)
	require.ErrorContains(t, s.err, expected.Error())
}

// TestInitializesBeforeProcessing verifies that callback initialization completes
// before the first batch reaches business processing.
func (s *BatchConsumerTestSuite) TestInitializesBeforeProcessing() {
	t := s.T()
	var initialized atomic.Bool
	processed := make(chan bool, 1)
	s.startConsumerWithCallback(1, time.Hour, &batchTestCallback{
		init: func(context.Context) error {
			initialized.Store(true)

			return nil
		},
		consume: func(ctx context.Context, models []any, attributes []map[string]string) ([]bool, error) {
			processed <- initialized.Load()

			return successfulBatch(ctx, models, attributes)
		},
	})
	s.publishModel(1)

	select {
	case ready := <-processed:
		require.True(t, ready)
	case <-time.After(time.Second):
		t.Fatal("initialized callback did not process work")
	}

	s.cancel()
	waitBatchTest(t, s.done)
	require.NoError(t, s.err)
}

// TestInitializationFailurePreventsBackgroundWork verifies that an initialization error
// is returned without starting the background callback or admitting primary messages.
func (s *BatchConsumerTestSuite) TestInitializationFailurePreventsBackgroundWork() {
	t := s.T()
	expected := errors.New("initialization failed")
	var ran atomic.Bool
	s.startConsumerWithCallback(1, time.Hour, &batchTestCallback{
		init:    func(context.Context) error { return expected },
		consume: successfulBatch,
		run: func(context.Context) error {
			ran.Store(true)

			return nil
		},
	})
	waitBatchTest(t, s.done)
	require.ErrorIs(t, s.err, expected)
	require.False(t, ran.Load())
	require.Empty(t, s.input.admitted)
}

type batchContextKey struct{}

type batchContextHandler struct{}

func (batchContextHandler) Encode(ctx context.Context, _ any, attributes map[string]string) (context.Context, map[string]string, error) {
	return ctx, attributes, nil
}

func (batchContextHandler) Decode(ctx context.Context, _ any, attributes map[string]string) (context.Context, map[string]string, error) {
	value := attributes["correlation"]
	delete(attributes, "correlation")

	return context.WithValue(ctx, batchContextKey{}, value), attributes, nil
}

// TestPreservesPropagationAttributesForRetries verifies that decoding restores callback context
// on both attempts without removing propagation attributes from the original message needed for retry.
func (s *BatchConsumerTestSuite) TestPreservesPropagationAttributesForRetries() {
	t := s.T()
	var calls atomic.Int32
	contexts := make(chan bool, 2)
	s.startConsumerWithCallback(1, time.Millisecond, &batchTestCallback{
		consume: func(ctx context.Context, models []any, attributes []map[string]string) ([]bool, error) {
			_, leaked := attributes[0]["correlation"]
			contexts <- ctx.Value(batchContextKey{}) == "record-42" && !leaked && models[0].(*batchTestModel).Value == 42

			return []bool{calls.Add(1) > 1}, nil
		},
	}, batchContextHandler{})
	msg, err := stream.MarshalJsonMessage(batchTestModel{Value: 42}, map[string]string{"correlation": "record-42"})
	require.NoError(t, err)
	s.input.Publish(msg)

	for range 2 {
		select {
		case propagated := <-contexts:
			require.True(t, propagated)
		case <-time.After(time.Second):
			t.Fatal("did not observe initial and retry processing")
		}
	}

	s.cancel()
	waitBatchTest(t, s.done)
	require.NoError(t, s.err)
	require.Equal(t, "record-42", msg.Attributes["correlation"])
}

// TestDisaggregatesAndRetriesMalformedMessages verifies that aggregate children are processed
// and malformed JSON does not stall a subsequent valid record; an unsuccessful drain reports replay needs.
func (s *BatchConsumerTestSuite) TestDisaggregatesAndRetriesMalformedMessages() {
	t := s.T()
	processed := make(chan int, 10)
	s.startConsumer(2, time.Millisecond, func(ctx context.Context, models []any, attributes []map[string]string) ([]bool, error) {
		for _, model := range models {
			processed <- model.(*batchTestModel).Value
		}

		return successfulBatch(ctx, models, attributes)
	})
	first, err := stream.MarshalJsonMessage(batchTestModel{Value: 1})
	require.NoError(t, err)
	second, err := stream.MarshalJsonMessage(batchTestModel{Value: 2})
	require.NoError(t, err)
	aggregate, err := stream.MarshalJsonMessage([]*stream.Message{first, second}, map[string]string{stream.AttributeAggregate: "true"})
	require.NoError(t, err)
	s.input.Publish(aggregate)

	for range 2 {
		select {
		case <-processed:
		case <-time.After(time.Second):
			t.Fatal("aggregate children were not processed")
		}
	}

	s.input.Publish(stream.NewMessage("not json", map[string]string{stream.AttributeEncoding: stream.EncodingJson.String()}))
	// A valid subsequent record still makes progress alongside decoder failures.
	s.publishModel(3)

	select {
	case value := <-processed:
		require.Equal(t, 3, value)
	case <-time.After(time.Second):
		t.Fatal("valid record stalled behind decoder failure")
	}

	s.cancel()
	waitBatchTest(t, s.done)

	// It either exhausted its retries already or remains unsuccessful in the drain.
	if s.err != nil {
		require.ErrorContains(t, s.err, "unsuccessful acknowledged messages requiring replay")
	}
}

// TestBatchConsumerFactoryRejectsUnknownRetryHandler verifies that factory construction
// rejects an enabled retry configuration whose handler type is not registered.
func TestBatchConsumerFactoryRejectsUnknownRetryHandler(t *testing.T) {
	config := cfg.New(map[string]any{"app": map[string]any{"name": "batch-test", "env": "test"}, "stream": map[string]any{
		"consumer": map[string]any{"test": map[string]any{
			"input": "batch-unknown-retry", "retry": map[string]any{"enabled": true, "type": "unknown"},
		}},
		"input": map[string]any{"batch-unknown-retry": map[string]any{"type": "inMemory"}},
	}})
	factory := stream.NewUntypedBatchConsumer("test", func(context.Context, cfg.Config, log.Logger) (stream.UntypedBatchConsumerCallback, error) {
		return &batchTestCallback{consume: successfulBatch}, nil
	})
	_, err := factory(appctx.WithContainer(t.Context()), config, logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t)))
	require.ErrorContains(t, err, "there is no retry handler of type unknown available")
}

// TestRetryAcknowledgementWaitsForProcessingWithSingleRunner verifies that a lone retry
// flushes below the batch-size threshold but is acknowledged only after business processing succeeds.
func (s *BatchConsumerTestSuite) TestRetryAcknowledgementWaitsForProcessingWithSingleRunner() {
	t := s.T()
	started, release := make(chan struct{}), make(chan struct{})
	defer close(release)
	s.startConsumer(100, time.Hour, func(ctx context.Context, models []any, attributes []map[string]string) ([]bool, error) {
		close(started)
		<-release

		return successfulBatch(ctx, models, attributes)
	})
	message, err := stream.MarshalJsonMessage(batchTestModel{Value: 1})
	require.NoError(t, err)
	s.retryInput.Publish(message)
	waitBatchTest(t, started)

	select {
	case <-s.retryInput.results:
		t.Fatal("retry acknowledged before business completion")
	default:
	}

	release <- struct{}{}
	select {
	case ack := <-s.retryInput.results:
		require.True(t, ack)
	case <-time.After(time.Second):
		t.Fatal("successful retry was not acknowledged")
	}

	s.cancel()
	waitBatchTest(t, s.done)
	require.NoError(t, s.err)
}

// TestFailedFinalRetryWriteRequiresReplay verifies that a failed callback during final draining,
// combined with an unavailable retry queue, returns an error identifying acknowledged work requiring replay.
func (s *BatchConsumerTestSuite) TestFailedFinalRetryWriteRequiresReplay() {
	t := s.T()
	s.retryWriteErr = errors.New("retry queue unavailable")
	s.startConsumer(100, time.Hour, func(context.Context, []any, []map[string]string) ([]bool, error) {
		return []bool{false}, errors.New("business failure")
	})
	s.publishModel(1)
	s.cancel()
	waitBatchTest(t, s.done)
	require.ErrorContains(t, s.err, "unsuccessful acknowledged messages requiring replay")
	require.Empty(t, s.retryMessages)
}

// TestFailedRetryWriteBeforeDrainingRequiresReplay verifies that failures from every
// collection path remain visible after subsequent work succeeds and the inputs drain.
func (s *BatchConsumerTestSuite) TestFailedRetryWriteBeforeDrainingRequiresReplay() {
	for _, tc := range []struct {
		name      string
		batchSize int
		interval  time.Duration
		malformed bool
		retry     bool
	}{
		{name: "size flush", batchSize: 1, interval: time.Hour},
		{name: "timer flush", batchSize: 100, interval: 5 * time.Millisecond},
		{name: "retry flush", batchSize: 100, interval: time.Hour, retry: true},
		{name: "malformed aggregate", batchSize: 1, interval: time.Hour, malformed: true},
	} {
		s.Run(tc.name, func() {
			t := s.T()
			s.retryWriteErr = errors.New("retry queue unavailable")
			processed := make(chan struct{}, 1)
			failed := make(chan struct{}, 1)
			s.startConsumer(tc.batchSize, tc.interval, func(_ context.Context, models []any, _ []map[string]string) ([]bool, error) {
				if models[0].(*batchTestModel).Value == 1 {
					failed <- struct{}{}

					return []bool{false}, errors.New("business failure")
				}

				processed <- struct{}{}

				return []bool{true}, nil
			})

			if tc.malformed {
				s.input.Publish(stream.NewMessage("not json", map[string]string{
					stream.AttributeEncoding:  stream.EncodingJson.String(),
					stream.AttributeAggregate: "true",
				}))
			} else {
				s.publishModel(1)
				if !tc.retry {
					waitBatchTest(t, failed)
				}
			}

			if tc.retry {
				message, err := stream.MarshalJsonMessage(batchTestModel{Value: 2})
				require.NoError(t, err)
				s.retryInput.Publish(message)
				select {
				case ack := <-s.retryInput.results:
					require.True(t, ack)
				case <-time.After(time.Second):
					t.Fatal("retry input did not receive success acknowledgement")
				}
			} else {
				s.publishModel(2)
			}

			waitBatchTest(t, processed)
			// The successful subsequent callback proves that the earlier failure was
			// handled by the collection loop before shutdown starts.
			s.cancel()
			waitBatchTest(t, s.done)
			require.ErrorContains(t, s.err, "1 unsuccessful acknowledged messages requiring replay")
			require.Empty(t, s.retryMessages)
		})
	}
}

// TestBatchConsumerWithInterfacesRejectsInvalidSettings verifies that zero or negative batch sizes
// and negative buffer sizes return an error and no consumer before accessing the supplied dependencies.
func TestBatchConsumerWithInterfacesRejectsInvalidSettings(t *testing.T) {
	for _, tc := range []struct {
		name     string
		settings stream.BatchConsumerSettings
	}{
		{name: "zero batch size", settings: stream.BatchConsumerSettings{BatchSize: 0}},
		{name: "negative batch size", settings: stream.BatchConsumerSettings{BatchSize: -1}},
		{name: "negative buffer size", settings: stream.BatchConsumerSettings{BatchSize: 1, BufferSize: -1}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			consumer, err := stream.NewUntypedBatchConsumerWithInterfaces(nil, nil, tc.settings)
			require.ErrorContains(t, err, "invalid batch consumer settings")
			require.Nil(t, consumer)
		})
	}
}

// TestBatchConsumerModuleFactoryNames verifies that each callback map entry produces
// a module factory named with the consumer- prefix.
func TestBatchConsumerModuleFactoryNames(t *testing.T) {
	modules, err := stream.BatchConsumerFactory(stream.UntypedBatchConsumerCallbackMap{"first": nil, "second": nil})
	require.NoError(t, err)
	require.Contains(t, modules, "consumer-first")
	require.Contains(t, modules, "consumer-second")
}

type typedBatchTestCallback struct{}

func (typedBatchTestCallback) Consume(_ context.Context, models []batchTestModel, _ []map[string]string) ([]bool, error) {
	acks := make([]bool, len(models))
	for i, model := range models {
		acks[i] = model.Value > 0
	}

	return acks, nil
}

// TestTypedBatchConsumerFactoryAndTypeErasure verifies that the typed factory builds a batch
// consumer and its untyped adapter creates the expected model and preserves per-record acknowledgements.
func TestTypedBatchConsumerFactoryAndTypeErasure(t *testing.T) {
	config := cfg.New(map[string]any{
		"app": map[string]any{"name": "batch-test", "env": "test"},
		"stream": map[string]any{
			"consumer": map[string]any{"test": map[string]any{"input": "batch-factory", "batch_size": 2}},
			"input":    map[string]any{"batch-factory": map[string]any{"type": "inMemory"}},
		},
	})
	factory := stream.NewTypedBatchConsumer("test", func(context.Context, cfg.Config, log.Logger) (stream.BatchConsumerCallback[batchTestModel], error) {
		return typedBatchTestCallback{}, nil
	})
	module, err := factory(appctx.WithContainer(t.Context()), config, logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t)))
	require.NoError(t, err)
	require.IsType(t, &stream.BatchConsumer{}, module)
	callback := stream.EraseBatchConsumerCallbackTypes[batchTestModel](typedBatchTestCallback{})
	model, err := callback.GetModel(nil)
	require.NoError(t, err)
	require.IsType(t, &batchTestModel{}, model)
	acks, err := callback.Consume(t.Context(), []any{&batchTestModel{Value: 1}, &batchTestModel{Value: 0}}, []map[string]string{nil, nil})
	require.NoError(t, err)
	require.Equal(t, []bool{true, false}, acks)
}

func (s *BatchConsumerTestSuite) startConsumer(size int, interval time.Duration, consume func(context.Context, []any, []map[string]string) ([]bool, error)) {
	s.T().Helper()
	s.startConsumerWithCallback(size, interval, &batchTestCallback{consume: consume})
}

func (s *BatchConsumerTestSuite) startConsumerWithCallback(size int, interval time.Duration, callback stream.UntypedBatchConsumerCallback, handlers ...stream.EncodeHandler) {
	t := s.T()
	t.Helper()
	require.Nil(t, s.consumer, "start only one consumer per fixture")
	s.settings.IdleTimeout = interval
	s.batchSettings.BatchSize = size

	base := stream.NewConsumerBaseWithInterfaces(
		uuid.New(),
		s.logger,
		s.metrics,
		tracing.NewLocalTracer(),
		s.input,
		stream.NewMessageEncoder(&stream.MessageEncoderSettings{Encoding: stream.EncodingJson, EncodeHandlers: handlers}),
		s.retryInput,
		stream.NewRetryHandlerSqsWithInterfaces(s.output, &stream.RetryHandlerSqsSettings{RetryHandlerSettings: stream.RetryHandlerSettings{After: time.Second, MaxAttempts: 3}}),
		s.settings,
		"test",
		nil,
		clock.NewRealClock(),
	)

	batch, err := stream.NewUntypedBatchConsumerWithInterfaces(base, callback, s.batchSettings)
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(t.Context())
	s.consumer, s.cancel = batch, cancel
	done := s.done

	go func() {
		s.err = batch.Run(ctx)
		close(done)
	}()
	t.Cleanup(func() {
		cancel()
		waitBatchTest(t, done)
	})
}

func (s *BatchConsumerTestSuite) publishModel(value int) {
	t := s.T()
	t.Helper()
	msg, err := stream.MarshalJsonMessage(batchTestModel{Value: value})
	require.NoError(t, err)
	s.input.Publish(msg)

	select {
	case ack := <-s.input.admitted:
		require.True(t, ack)
	case <-time.After(time.Second):
		t.Fatal("input callback did not acknowledge buffering")
	}
}

type batchTestModel struct {
	Value int `json:"value"`
}

type batchTestCallback struct {
	consume func(context.Context, []any, []map[string]string) ([]bool, error)
	init    func(context.Context) error
	run     func(context.Context) error
}

func (b *batchTestCallback) Init(ctx context.Context) error {
	if b.init != nil {
		return b.init(ctx)
	}

	return nil
}

func (*batchTestCallback) GetModel(map[string]string) (any, error) {
	return new(batchTestModel), nil
}

func (b *batchTestCallback) Consume(ctx context.Context, models []any, attributes []map[string]string) ([]bool, error) {
	return b.consume(ctx, models, attributes)
}

func (b *batchTestCallback) Run(ctx context.Context) error {
	if b.run != nil {
		return b.run(ctx)
	}

	return nil
}

type batchTestInput struct {
	*stream.InMemoryInput
	admitted chan bool
}

func (i *batchTestInput) Run(ctx context.Context, process stream.InputProcess) error {
	return i.InMemoryInput.Run(ctx, func(ctx context.Context, msg *stream.Message) bool {
		ack := process(ctx, msg)
		i.admitted <- ack

		return ack
	})
}

// Simulate SQS redelivery and a three-receive redrive limit without AWS.
type batchTestRetryInput struct {
	*stream.InMemoryInput
	results chan bool
}

func (i *batchTestRetryInput) Run(ctx context.Context, process stream.InputProcess) error {
	return i.InMemoryInput.Run(ctx, func(ctx context.Context, msg *stream.Message) bool {
		for range 3 {
			ack := process(ctx, msg)
			i.results <- ack

			if ack {
				return true
			}
		}

		return false
	})
}

func waitBatchTest(t *testing.T, done <-chan struct{}) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("batch consumer did not finish")
	}
}

func successfulBatch(_ context.Context, models []any, _ []map[string]string) ([]bool, error) {
	acks := make([]bool, len(models))
	for i := range acks {
		acks[i] = true
	}

	return acks, nil
}
