package stream_test

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/encoding/json"
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/justtrackio/gosoline/pkg/log"
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
	"github.com/stretchr/testify/suite"
)

func TestConsumerTestSuite(t *testing.T) {
	suite.Run(t, new(ConsumerTestSuite))
}

type ConsumerTestSuite struct {
	suite.Suite

	kernelCtx    context.Context
	kernelCancel context.CancelFunc

	input         *mocks.Input
	inputData     chan *stream.Message
	inputDataOut  <-chan *stream.Message
	inputStopOnce sync.Once
	inputStop     func(context.Context)

	retryInput    *mocks.Input
	retryHandler  *mocks.RetryHandler
	retryData     chan *stream.Message
	retryDataOut  <-chan *stream.Message
	retryStopOnce sync.Once
	retryStop     func(context.Context)

	uuidGen  *uuidMocks.Uuid
	callback *mocks.RunnableUntypedConsumerCallback
	consumer *stream.Consumer
	clock    clock.FakeClock

	settings        stream.ConsumerSettings
	logger          log.Logger
	tracer          tracing.Tracer
	metricWriter    *metricMocks.Writer
	encoder         stream.MessageEncoder
	samplingDecider *smplMocks.Decider
}

func (s *ConsumerTestSuite) SetupTest() {
	s.kernelCtx, s.kernelCancel = context.WithCancel(s.T().Context())

	s.inputData = make(chan *stream.Message, 10)
	s.inputDataOut = s.inputData
	s.inputStopOnce = sync.Once{}
	s.inputStop = func(context.Context) {}

	s.input = mocks.NewInput(s.T())
	s.input.EXPECT().Stop(matcher.Context).Run(func(ctx context.Context) {
		s.NoError(ctx.Err())
		s.inputStop(ctx)
	}).Once()

	s.retryData = make(chan *stream.Message, 10)
	s.retryDataOut = s.retryData
	s.retryStopOnce = sync.Once{}
	s.retryStop = func(context.Context) {
		s.retryStopOnce.Do(func() {
			close(s.retryData)
		})
	}

	s.retryInput = mocks.NewInput(s.T())
	s.retryInput.EXPECT().Stop(matcher.Context).Run(func(ctx context.Context) {
		s.NoError(ctx.Err())
		s.retryStop(ctx)
	}).Once()

	s.retryHandler = mocks.NewRetryHandler(s.T())

	s.uuidGen = uuidMocks.NewUuid(s.T())
	s.callback = mocks.NewRunnableUntypedConsumerCallback(s.T())

	s.logger = logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(s.T()))
	s.tracer = tracing.NewLocalTracer()
	s.metricWriter = metricMocks.NewWriter(s.T())
	s.metricWriter.EXPECT().Write(matcher.Context, mock.Anything).Return().Maybe()
	s.encoder = stream.NewMessageEncoder(&stream.MessageEncoderSettings{})

	s.settings = stream.ConsumerSettings{
		Input:       "test",
		IdleTimeout: time.Second,
		Retry: stream.ConsumerRetrySettings{
			Enabled: true,
		},
		Healthcheck: health.HealthCheckSettings{
			Timeout: time.Minute,
		},
		AggregateMessageMode: stream.AggregateMessageModeAtMostOnce,
	}

	s.clock = clock.NewFakeClock()

	s.samplingDecider = smplMocks.NewDecider(s.T())
	s.samplingDecider.EXPECT().Decide(matcher.Context).RunAndReturn(func(ctx context.Context, strategy ...smpl.Strategy) (context.Context, bool, error) {
		return ctx, false, nil
	}).Maybe()

	s.buildConsumer()
}

// buildConsumer wires a consumer from the current settings. Tests which need different settings than the defaults
// from SetupTest can adjust s.settings and call this again before running the consumer.
func (s *ConsumerTestSuite) buildConsumer() {
	s.consumer = stream.NewUntypedConsumerWithInterfaces(
		s.uuidGen,
		s.logger,
		s.metricWriter,
		s.tracer,
		s.input,
		s.encoder,
		s.retryInput,
		s.retryHandler,
		s.callback,
		s.settings,
		"test",
		s.samplingDecider,
		s.clock,
	)
}

func (s *ConsumerTestSuite) expectInputRun(messages ...*stream.Message) {
	s.input.EXPECT().
		Run(matcher.Context, mock.Anything).
		RunAndReturn(func(ctx context.Context, process stream.InputProcess) error {
			for _, msg := range messages {
				process(ctx, msg)
			}

			return nil
		}).
		Once()
}

func (s *ConsumerTestSuite) expectInputRunUntilStopped() {
	s.input.EXPECT().
		Run(matcher.Context, mock.Anything).
		RunAndReturn(func(ctx context.Context, _ stream.InputProcess) error {
			<-ctx.Done()

			return nil
		}).
		Once()
}

func (s *ConsumerTestSuite) expectRetryInputRun() {
	s.retryInput.EXPECT().
		Run(matcher.Context, mock.Anything).
		RunAndReturn(func(ctx context.Context, process stream.InputProcess) error {
			for {
				select {
				case <-ctx.Done():
					return nil
				case msg, ok := <-s.retryData:
					if !ok {
						return nil
					}

					process(ctx, msg)
				}
			}
		}).
		Once()
}

func (s *ConsumerTestSuite) expectRetryInputRunAndInject(msg *stream.Message) {
	s.retryInput.EXPECT().
		Run(matcher.Context, mock.Anything).
		RunAndReturn(func(ctx context.Context, process stream.InputProcess) error {
			process(ctx, msg)

			return nil
		}).
		Once()
}

func (s *ConsumerTestSuite) TestGetModelNil() {
	s.expectRetryInputRun()
	s.expectInputRun(stream.NewJsonMessage(`"foo"`, map[string]string{
		"bla": "blub",
	}))

	s.callback.EXPECT().GetModel(mock.AnythingOfType("map[string]string")).Return(nil, nil).Once()
	s.callback.EXPECT().Run(matcher.Context).Return(nil).Once()

	err := s.consumer.Run(s.kernelCtx)

	s.NoError(err, "there should be no error during run")
}

func (s *ConsumerTestSuite) TestRun() {
	s.expectRetryInputRun()
	s.expectInputRun(
		stream.NewJsonMessage(`"foo"`),
		stream.NewJsonMessage(`"bar"`),
		stream.NewJsonMessage(`"foobar"`),
	)

	consumed := make([]*string, 0)

	s.callback.EXPECT().
		Consume(matcher.Context, mock.AnythingOfType("*string"), map[string]string{
			stream.AttributeEncoding: stream.EncodingJson.String(),
		}).
		Run(func(ctx context.Context, model any, attributes map[string]string) {
			consumed = append(consumed, model.(*string))
			if len(consumed) == 3 {
				s.kernelCancel()
			}
		}).
		Return(true, nil).
		Times(3)

	s.callback.EXPECT().
		GetModel(mock.AnythingOfType("map[string]string")).
		Return(mdl.Box(""), nil).
		Times(3)

	s.callback.EXPECT().Run(matcher.Context).Return(nil).Once()

	err := s.consumer.Run(s.kernelCtx)

	s.NoError(err, "there should be no error during run")
	s.Len(consumed, 3)
}

func (s *ConsumerTestSuite) TestIsHealthyWhileInputIsIdle() {
	s.input.EXPECT().IsHealthy().Return(true).Once()
	s.retryInput.EXPECT().IsHealthy().Return(true).Once()

	s.clock.Advance(2 * time.Minute)

	healthy, err := s.consumer.IsHealthy(s.T().Context())

	s.NoError(err)
	s.True(healthy)

	s.input.Stop(s.T().Context())
	s.retryInput.Stop(s.T().Context())
}

func (s *ConsumerTestSuite) TestIsUnhealthyWhileProcessingIsStuck() {
	msg := stream.NewJsonMessage(`"foo"`)
	processingStarted := make(chan struct{})
	releaseProcessing := make(chan struct{})

	s.expectInputRun(msg)
	s.expectRetryInputRun()
	s.callback.EXPECT().Run(matcher.Context).RunAndReturn(func(ctx context.Context) error {
		<-ctx.Done()

		return nil
	}).Once()
	s.callback.EXPECT().GetModel(mock.AnythingOfType("map[string]string")).Return(mdl.Box(""), nil).Once()
	s.callback.EXPECT().Consume(matcher.Context, mock.AnythingOfType("*string"), map[string]string{
		stream.AttributeEncoding: stream.EncodingJson.String(),
	}).RunAndReturn(func(context.Context, any, map[string]string) (bool, error) {
		close(processingStarted)
		<-releaseProcessing

		return true, nil
	}).Once()
	s.input.EXPECT().IsHealthy().Return(true).Once()
	s.retryInput.EXPECT().IsHealthy().Return(true).Once()

	runDone := make(chan error, 1)
	go func() {
		runDone <- s.consumer.Run(s.kernelCtx)
	}()

	<-processingStarted
	s.clock.Advance(2 * time.Minute)

	healthy, err := s.consumer.IsHealthy(s.T().Context())
	s.NoError(err)
	s.False(healthy)

	close(releaseProcessing)
	s.kernelCancel()

	select {
	case err = <-runDone:
		s.NoError(err)
	case <-time.After(time.Second):
		s.FailNow("consumer did not finish after processing resumed")
	}
}

func (s *ConsumerTestSuite) TestIsHealthyWhileProcessingContinuously() {
	firstMsg := stream.NewJsonMessage(`"first"`)
	secondMsg := stream.NewJsonMessage(`"second"`)
	firstStarted := make(chan struct{})
	secondStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	releaseSecond := make(chan struct{})

	s.expectInputRun(firstMsg, secondMsg)
	s.expectRetryInputRun()
	s.callback.EXPECT().Run(matcher.Context).RunAndReturn(func(ctx context.Context) error {
		<-ctx.Done()

		return nil
	}).Once()
	s.callback.EXPECT().GetModel(mock.AnythingOfType("map[string]string")).RunAndReturn(func(map[string]string) (any, error) {
		return mdl.Box(""), nil
	}).Twice()
	s.callback.EXPECT().Consume(matcher.Context, mock.AnythingOfType("*string"), map[string]string{
		stream.AttributeEncoding: stream.EncodingJson.String(),
	}).RunAndReturn(func(_ context.Context, model any, _ map[string]string) (bool, error) {
		switch *model.(*string) {
		case "first":
			close(firstStarted)
			<-releaseFirst
		case "second":
			close(secondStarted)
			<-releaseSecond
		}

		return true, nil
	}).Twice()
	s.input.EXPECT().IsHealthy().Return(true).Once()
	s.retryInput.EXPECT().IsHealthy().Return(true).Once()

	runDone := make(chan error, 1)
	go func() {
		runDone <- s.consumer.Run(s.kernelCtx)
	}()

	<-firstStarted
	s.clock.Advance(30 * time.Second)
	close(releaseFirst)
	<-secondStarted
	s.clock.Advance(40 * time.Second)

	healthy, err := s.consumer.IsHealthy(s.T().Context())
	s.NoError(err)
	s.True(healthy)

	close(releaseSecond)
	s.kernelCancel()

	select {
	case err = <-runDone:
		s.NoError(err)
	case <-time.After(time.Second):
		s.FailNow("consumer did not finish after processing resumed")
	}
}

func (s *ConsumerTestSuite) TestIsHealthyWhileProcessingAggregateContinuously() {
	firstMsg := stream.NewJsonMessage(`"first"`)
	secondMsg := stream.NewJsonMessage(`"second"`)
	aggregateBody, err := json.Marshal([]stream.WritableMessage{firstMsg, secondMsg})
	s.NoError(err)
	aggregate := stream.BuildAggregateMessage(string(aggregateBody))
	firstStarted := make(chan struct{})
	secondStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	releaseSecond := make(chan struct{})

	s.expectInputRun(aggregate)
	s.expectRetryInputRun()
	s.callback.EXPECT().Run(matcher.Context).RunAndReturn(func(ctx context.Context) error {
		<-ctx.Done()

		return nil
	}).Once()
	s.callback.EXPECT().GetModel(mock.AnythingOfType("map[string]string")).RunAndReturn(func(map[string]string) (any, error) {
		return mdl.Box(""), nil
	}).Twice()
	s.callback.EXPECT().Consume(matcher.Context, mock.AnythingOfType("*string"), map[string]string{
		stream.AttributeEncoding: stream.EncodingJson.String(),
	}).RunAndReturn(func(_ context.Context, model any, _ map[string]string) (bool, error) {
		switch *model.(*string) {
		case "first":
			close(firstStarted)
			<-releaseFirst
		case "second":
			close(secondStarted)
			<-releaseSecond
		}

		return true, nil
	}).Twice()
	s.input.EXPECT().IsHealthy().Return(true).Once()
	s.retryInput.EXPECT().IsHealthy().Return(true).Once()

	runDone := make(chan error, 1)
	go func() {
		runDone <- s.consumer.Run(s.kernelCtx)
	}()

	<-firstStarted
	s.clock.Advance(30 * time.Second)
	close(releaseFirst)
	<-secondStarted
	s.clock.Advance(40 * time.Second)

	healthy, err := s.consumer.IsHealthy(s.T().Context())
	s.NoError(err)
	s.True(healthy)

	close(releaseSecond)
	s.kernelCancel()

	select {
	case err = <-runDone:
		s.NoError(err)
	case <-time.After(time.Second):
		s.FailNow("consumer did not finish after aggregate processing resumed")
	}
}

func (s *ConsumerTestSuite) TestIsUnhealthyWhenConcurrentProcessingIsStuck() {
	stuckMsg := stream.NewJsonMessage(`"stuck"`)
	flowingMsg := stream.NewJsonMessage(`"flowing"`)
	stuckStarted := make(chan struct{})
	releaseStuck := make(chan struct{})
	flowingDone := make(chan struct{})
	inputStopped := make(chan struct{})

	s.inputStop = func(context.Context) {
		close(inputStopped)
	}

	// Both messages are processed concurrently in their own goroutine. They have to be awaited before Run returns,
	// otherwise they can outlive the test and call Ack on the mock of the test which runs next.
	var processing sync.WaitGroup
	processing.Add(2)

	s.input.EXPECT().
		Run(matcher.Context, mock.Anything).
		RunAndReturn(func(ctx context.Context, process stream.InputProcess) error {
			go func() {
				defer processing.Done()

				process(ctx, stuckMsg)
			}()
			go func() {
				defer processing.Done()

				process(ctx, flowingMsg)
			}()

			<-inputStopped
			processing.Wait()

			return nil
		}).
		Once()
	s.expectRetryInputRun()
	s.callback.EXPECT().Run(matcher.Context).RunAndReturn(func(ctx context.Context) error {
		<-ctx.Done()

		return nil
	}).Once()
	s.callback.EXPECT().GetModel(mock.AnythingOfType("map[string]string")).RunAndReturn(func(map[string]string) (any, error) {
		return mdl.Box(""), nil
	}).Twice()
	s.callback.EXPECT().Consume(matcher.Context, mock.AnythingOfType("*string"), map[string]string{
		stream.AttributeEncoding: stream.EncodingJson.String(),
	}).RunAndReturn(func(_ context.Context, model any, _ map[string]string) (bool, error) {
		switch *model.(*string) {
		case "stuck":
			close(stuckStarted)
			<-releaseStuck
		case "flowing":
			close(flowingDone)
		}

		return true, nil
	}).Twice()
	s.input.EXPECT().IsHealthy().Return(true).Once()
	s.retryInput.EXPECT().IsHealthy().Return(true).Once()

	runDone := make(chan error, 1)
	go func() {
		runDone <- s.consumer.Run(s.kernelCtx)
	}()

	<-stuckStarted
	<-flowingDone
	s.clock.Advance(2 * time.Minute)

	healthy, err := s.consumer.IsHealthy(s.T().Context())
	s.NoError(err)
	s.False(healthy)

	close(releaseStuck)
	s.kernelCancel()

	select {
	case err = <-runDone:
		s.NoError(err)
	case <-time.After(time.Second):
		s.FailNow("consumer did not finish after processing resumed")
	}
}

func (s *ConsumerTestSuite) TestRun_ProcessesMessageFetchedBeforeShutdown() {
	msg := stream.NewJsonMessage(`"foo"`)
	inputStopped := make(chan struct{})
	processed := make(chan struct{})

	// Keep the grace window open while the input hands off a message from its already-fetched batch.
	s.settings.GraceTime = time.Minute
	s.buildConsumer()

	s.inputStop = func(context.Context) {
		close(inputStopped)
	}
	s.input.EXPECT().
		Run(matcher.Context, mock.Anything).
		RunAndReturn(func(ctx context.Context, process stream.InputProcess) error {
			<-inputStopped

			process(ctx, msg)

			return nil
		}).
		Once()
	s.expectRetryInputRun()
	s.callback.EXPECT().Run(matcher.Context).RunAndReturn(func(ctx context.Context) error {
		<-ctx.Done()

		return nil
	}).Once()
	s.callback.EXPECT().GetModel(mock.AnythingOfType("map[string]string")).Return(mdl.Box(""), nil).Once()
	s.callback.EXPECT().Consume(matcher.Context, mock.AnythingOfType("*string"), map[string]string{
		stream.AttributeEncoding: stream.EncodingJson.String(),
	}).Run(func(context.Context, any, map[string]string) {
		close(processed)
	}).Return(true, nil).Once()

	runDone := make(chan error, 1)
	go func() {
		runDone <- s.consumer.Run(s.kernelCtx)
	}()

	s.kernelCancel()

	select {
	case <-processed:
	case <-time.After(time.Second):
		s.FailNow("message fetched before shutdown was not processed during the grace window")
	}

	select {
	case err := <-runDone:
		s.NoError(err)
	case <-time.After(time.Second):
		s.FailNow("consumer did not finish after draining the fetched message")
	}
}

func (s *ConsumerTestSuite) TestRun_WaitsForCanceledInFlightProcessingBeforeStoppingCallback() {
	msg := stream.NewJsonMessage(`"foo"`)
	callbackStarted := make(chan struct{})
	processingStarted := make(chan struct{})
	inputStopped := make(chan struct{})
	callbackStopped := make(chan struct{})
	processingCanceled := make(chan struct{})
	processCtx, cancelProcess := context.WithCancel(s.T().Context())

	s.inputStop = func(context.Context) {
		cancelProcess()
		close(inputStopped)
	}
	s.input.EXPECT().
		Run(matcher.Context, mock.Anything).
		RunAndReturn(func(_ context.Context, process stream.InputProcess) error {
			process(processCtx, msg)

			return nil
		}).
		Once()
	s.expectRetryInputRun()

	s.callback.EXPECT().Run(matcher.Context).RunAndReturn(func(ctx context.Context) error {
		close(callbackStarted)
		<-ctx.Done()

		close(callbackStopped)

		return nil
	}).Once()
	s.callback.EXPECT().GetModel(mock.AnythingOfType("map[string]string")).Return(mdl.Box(""), nil).Once()
	s.callback.EXPECT().Consume(matcher.Context, mock.AnythingOfType("*string"), map[string]string{
		stream.AttributeEncoding: stream.EncodingJson.String(),
	}).RunAndReturn(func(ctx context.Context, _ any, _ map[string]string) (bool, error) {
		close(processingStarted)
		<-ctx.Done()
		close(processingCanceled)

		return true, nil
	}).Once()

	runDone := make(chan error, 1)
	go func() {
		runDone <- s.consumer.Run(s.kernelCtx)
	}()

	select {
	case <-processingStarted:
	case <-callbackStopped:
		s.FailNow("callback stopped before message processing started")
	case <-inputStopped:
		s.FailNow("input stopped before message processing started")
	case <-time.After(time.Second):
		s.FailNow("message processing did not start")
	}
	<-callbackStarted
	s.kernelCancel()
	<-inputStopped

	select {
	case <-processingCanceled:
	case <-callbackStopped:
		s.Fail("callback stopped while message processing was still in flight")
	case <-time.After(time.Second):
		s.FailNow("processing context was not canceled during shutdown")
	}

	select {
	case err := <-runDone:
		s.NoError(err)
	case <-time.After(time.Second):
		s.FailNow("consumer did not finish shutting down")
	}
}

func (s *ConsumerTestSuite) TestRun_InputStopCancelsInFlightProcessing() {
	msg := stream.NewJsonMessage(`"foo"`)
	processingStarted := make(chan struct{})
	processingCanceled := make(chan struct{})
	processCtx, cancelProcess := context.WithCancel(s.T().Context())

	s.inputStop = func(context.Context) {
		cancelProcess()
	}
	s.input.EXPECT().
		Run(matcher.Context, mock.Anything).
		RunAndReturn(func(_ context.Context, process stream.InputProcess) error {
			process(processCtx, msg)

			return nil
		}).
		Once()
	s.expectRetryInputRun()
	s.callback.EXPECT().Run(matcher.Context).RunAndReturn(func(ctx context.Context) error {
		<-ctx.Done()

		return nil
	}).Once()
	s.callback.EXPECT().GetModel(mock.AnythingOfType("map[string]string")).Return(mdl.Box(""), nil).Once()
	s.callback.EXPECT().Consume(matcher.Context, mock.AnythingOfType("*string"), map[string]string{
		stream.AttributeEncoding: stream.EncodingJson.String(),
	}).RunAndReturn(func(ctx context.Context, _ any, _ map[string]string) (bool, error) {
		close(processingStarted)
		<-ctx.Done()
		close(processingCanceled)

		return true, nil
	}).Once()

	runDone := make(chan error, 1)
	go func() {
		runDone <- s.consumer.Run(s.kernelCtx)
	}()

	<-processingStarted
	s.kernelCancel()

	select {
	case <-processingCanceled:
	case <-time.After(time.Second):
		s.FailNow("input stop did not cancel in-flight processing")
	}

	select {
	case err := <-runDone:
		s.NoError(err)
	case <-time.After(time.Second):
		s.FailNow("consumer did not finish after in-flight processing was canceled")
	}
}

func (s *ConsumerTestSuite) TestRun_InputRunError() {
	s.expectRetryInputRun()
	s.input.EXPECT().
		Run(matcher.Context, mock.Anything).
		Return(fmt.Errorf("read error")).
		Once()

	s.callback.EXPECT().
		Run(matcher.Context).
		Run(func(ctx context.Context) {
			<-ctx.Done()
		}).
		Return(nil).
		Once()

	err := s.consumer.Run(s.kernelCtx)

	s.EqualError(err, "error while waiting for all routines to stop: panic during run of the consumer input: read error")
}

func (s *ConsumerTestSuite) TestRun_CallbackRunError() {
	s.expectRetryInputRun()
	s.expectInputRunUntilStopped()

	s.callback.EXPECT().
		Run(matcher.Context).
		Return(fmt.Errorf("consumerCallback run error")).
		Once()

	err := s.consumer.Run(s.T().Context())

	s.EqualError(err, "error while waiting for all routines to stop: panic during run of the consumerCallback: consumerCallback run error")
}

func (s *ConsumerTestSuite) TestRun_CallbackRunPanic() {
	consumed := make([]*string, 0)

	s.callback.EXPECT().Run(matcher.Context).Return(nil).Once()

	s.callback.EXPECT().
		Consume(matcher.Context, mock.AnythingOfType("*string"), map[string]string{
			stream.AttributeEncoding: stream.EncodingJson.String(),
		}).
		Run(func(ctx context.Context, model any, attributes map[string]string) {
			ptr := model.(*string)
			consumed = append(consumed, ptr)

			msg := *ptr
			if msg == "bar" {
				s.kernelCancel()
				panic("bar")
			}
		}).
		Return(true, nil).
		Twice()
	s.callback.EXPECT().
		GetModel(mock.AnythingOfType("map[string]string")).
		Return(mdl.Box(""), nil).
		Twice()

	retryMsg := &stream.Message{
		Attributes: map[string]string{
			stream.AttributeEncoding: stream.EncodingJson.String(),
			stream.AttributeRetry:    "true",
			stream.AttributeRetryId:  "75828fe1-4c7d-4a21-99e5-03d63876ed23",
		},
		Body: `"bar"`,
	}

	s.uuidGen.EXPECT().NewV4().Return("75828fe1-4c7d-4a21-99e5-03d63876ed23").Once()
	s.expectRetryInputRun()
	s.retryHandler.EXPECT().Put(matcher.Context, retryMsg).Return(nil).Once()
	s.expectInputRun(
		stream.NewJsonMessage(`"foo"`),
		stream.NewJsonMessage(`"bar"`),
	)

	err := s.consumer.Run(s.kernelCtx)

	s.Nil(err, "there should be no error returned on consume")
	s.Len(consumed, 2)
}

func (s *ConsumerTestSuite) TestRun_AggregateMessage() {
	s.settings.AggregateMessageMode = stream.AggregateMessageModeAtLeastOnce
	s.buildConsumer()
	s.expectRetryInputRun()

	message1 := stream.NewJsonMessage(`"foo"`, map[string]string{
		"attr1": "a",
	})
	message2 := stream.NewJsonMessage(`"bar"`, map[string]string{
		"attr1": "b",
	})

	aggregateBody, err := json.Marshal([]stream.WritableMessage{message1, message2})
	s.NoError(err)

	aggregate := stream.BuildAggregateMessage(string(aggregateBody))

	s.expectInputRun(aggregate)

	consumed := make([]string, 0)
	s.callback.EXPECT().Run(matcher.Context).Return(nil).Once()

	expectedAttributes1 := map[string]string{
		stream.AttributeEncoding: stream.EncodingJson.String(),
		"attr1":                  "a",
	}

	s.callback.EXPECT().
		Consume(matcher.Context, mock.AnythingOfType("*string"), expectedAttributes1).
		Run(func(ctx context.Context, model any, attributes map[string]string) {
			ptr := model.(*string)
			consumed = append(consumed, *ptr)

			if len(consumed) == 2 {
				s.kernelCancel()
			}
		}).
		Return(true, nil).
		Once()

	expectedModelAttributes1 := map[string]string{"attr1": "a", "encoding": "application/json"}
	s.callback.EXPECT().
		GetModel(expectedModelAttributes1).
		Return(mdl.Box(""), nil).
		Once()

	expectedAttributes2 := map[string]string{
		stream.AttributeEncoding: stream.EncodingJson.String(),
		"attr1":                  "b",
	}
	s.callback.EXPECT().
		Consume(matcher.Context, mock.AnythingOfType("*string"), expectedAttributes2).
		Run(func(ctx context.Context, model any, attributes map[string]string) {
			ptr := model.(*string)
			consumed = append(consumed, *ptr)

			if len(consumed) == 2 {
				s.kernelCancel()
			}
		}).
		Return(true, nil).
		Once()

	expectedModelAttributes2 := map[string]string{"attr1": "b", "encoding": "application/json"}
	s.callback.EXPECT().
		GetModel(expectedModelAttributes2).
		Return(mdl.Box(""), nil).
		Once()

	err = s.consumer.Run(s.kernelCtx)

	s.Nil(err, "there should be no error returned on consume")
	s.Len(consumed, 2)
	s.Equal("foobar", strings.Join(consumed, ""))
}

func (s *ConsumerTestSuite) TestRun_AggregateMessageAtMostOnceDoesNotAckWhenAllMessagesFail() {
	s.settings.Retry.Enabled = false
	s.buildConsumer()
	s.expectRetryInputRun()

	aggregate := stream.BuildAggregateMessage(`[{"body":"\"foo\"","attributes":{"encoding":"application/json"}}]`)
	s.expectInputRun(aggregate)

	s.callback.EXPECT().Run(matcher.Context).Return(nil).Once()
	s.callback.EXPECT().
		GetModel(mock.AnythingOfType("map[string]string")).
		Return(mdl.Box(""), nil).
		Once()
	s.callback.EXPECT().
		Consume(matcher.Context, mock.AnythingOfType("*string"), map[string]string{
			stream.AttributeEncoding: stream.EncodingJson.String(),
		}).
		Return(false, nil).
		Once()

	s.NoError(s.consumer.Run(s.kernelCtx))
}

func (s *ConsumerTestSuite) TestRun_AggregateMessageAtMostOnceAcksWhenAnyMessageSucceeds() {
	s.settings.Retry.Enabled = false
	s.buildConsumer()
	s.expectRetryInputRun()

	aggregate := stream.BuildAggregateMessage(`[{"body":"\"foo\"","attributes":{"encoding":"application/json"}},{"body":"\"bar\"","attributes":{"encoding":"application/json"}}]`)
	s.expectInputRun(aggregate)

	s.callback.EXPECT().Run(matcher.Context).Return(nil).Once()
	s.callback.EXPECT().
		GetModel(mock.AnythingOfType("map[string]string")).
		Return(mdl.Box(""), nil).
		Twice()
	s.callback.EXPECT().
		Consume(matcher.Context, mock.AnythingOfType("*string"), map[string]string{
			stream.AttributeEncoding: stream.EncodingJson.String(),
		}).
		Return(false, nil).
		Once()
	s.callback.EXPECT().
		Consume(matcher.Context, mock.AnythingOfType("*string"), map[string]string{
			stream.AttributeEncoding: stream.EncodingJson.String(),
		}).
		Return(true, nil).
		Once()

	s.NoError(s.consumer.Run(s.kernelCtx))
}

func (s *ConsumerTestSuite) TestRun_AggregateMessageAtLeastOnceDoesNotAckWhenAnyMessageFails() {
	s.settings.AggregateMessageMode = stream.AggregateMessageModeAtLeastOnce
	s.settings.Retry.Enabled = false
	s.buildConsumer()
	s.expectRetryInputRun()

	aggregate := stream.BuildAggregateMessage(`[{"body":"\"foo\"","attributes":{"encoding":"application/json"}},{"body":"\"bar\"","attributes":{"encoding":"application/json"}}]`)
	s.expectInputRun(aggregate)

	s.callback.EXPECT().Run(matcher.Context).Return(nil).Once()
	s.callback.EXPECT().
		GetModel(mock.AnythingOfType("map[string]string")).
		Return(mdl.Box(""), nil).
		Twice()
	s.callback.EXPECT().
		Consume(matcher.Context, mock.AnythingOfType("*string"), map[string]string{
			stream.AttributeEncoding: stream.EncodingJson.String(),
		}).
		Return(false, nil).
		Once()
	s.callback.EXPECT().
		Consume(matcher.Context, mock.AnythingOfType("*string"), map[string]string{
			stream.AttributeEncoding: stream.EncodingJson.String(),
		}).
		Return(true, nil).
		Once()

	s.NoError(s.consumer.Run(s.kernelCtx))
}

func (s *ConsumerTestSuite) TestRunWithRetry() {
	// The retry message is only queued once the consumer input already returned, so it gets processed while the
	// consumer is shutting down. Without a drain window the shared drain context is cancelled immediately and the
	// message would be bounced straight back into the retry handler instead of being consumed. The fake clock never
	// advances, so this grace time simply never expires during the test.
	s.settings.GraceTime = time.Minute
	s.buildConsumer()

	uuid := "243da976-c43f-4578-9307-596146e7dd9a"
	s.uuidGen.EXPECT().NewV4().Return(uuid)

	originalMessage := stream.NewJsonMessage(`"foo"`)
	retryMessage := stream.NewMessage(`"foo"`, map[string]string{
		stream.AttributeEncoding: stream.EncodingJson.String(),
		stream.AttributeRetry:    "true",
		stream.AttributeRetryId:  uuid,
	})

	s.expectInputRun(originalMessage)

	s.retryHandler.EXPECT().
		Put(matcher.Context, retryMessage).
		Run(func(ctx context.Context, msg *stream.Message) {
			s.retryData <- stream.NewMessage(`"foo from retry"`, map[string]string{
				stream.AttributeEncoding: stream.EncodingJson.String(),
				stream.AttributeRetry:    "true",
				stream.AttributeRetryId:  uuid,
			})
		}).
		Return(nil).
		Once()
	s.expectRetryInputRun()

	consumed := make([]string, 0)

	s.callback.EXPECT().
		Consume(matcher.Context, mock.AnythingOfType("*string"), map[string]string{
			stream.AttributeEncoding: stream.EncodingJson.String(),
		}).
		Run(func(ctx context.Context, model any, attributes map[string]string) {
			consumed = append(consumed, *model.(*string))
		}).
		Return(false, nil).
		Once()
	s.callback.EXPECT().
		Consume(matcher.Context, mock.AnythingOfType("*string"), map[string]string{
			stream.AttributeEncoding: stream.EncodingJson.String(),
			stream.AttributeRetry:    "true",
			stream.AttributeRetryId:  uuid,
		}).
		Run(func(ctx context.Context, model any, attributes map[string]string) {
			consumed = append(consumed, *model.(*string))
			s.kernelCancel()
		}).
		Return(true, nil).
		Once()

	s.callback.EXPECT().
		GetModel(mock.AnythingOfType("map[string]string")).
		Return(mdl.Box(""), nil).
		Twice()

	s.callback.EXPECT().Run(matcher.Context).Return(nil)

	err := s.consumer.Run(s.kernelCtx)

	s.NoError(err, "there should be no error during run")
	s.Equal("foo", consumed[0])
	s.Equal("foo from retry", consumed[1])
}

func (s *ConsumerTestSuite) TestRun_RetryProcessingIsCanceledWhenDrainGraceTimeExpires() {
	s.settings.GraceTime = time.Second
	s.buildConsumer()

	retryMessage := stream.NewJsonMessage(`"retry"`, map[string]string{
		stream.AttributeRetry: "true",
	})
	processingStarted := make(chan struct{})
	processingCanceled := make(chan struct{})

	s.expectInputRun()
	s.expectRetryInputRunAndInject(retryMessage)
	s.callback.EXPECT().Run(matcher.Context).RunAndReturn(func(ctx context.Context) error {
		<-ctx.Done()

		return nil
	}).Once()
	s.callback.EXPECT().GetModel(mock.AnythingOfType("map[string]string")).Return(mdl.Box(""), nil).Once()
	s.callback.EXPECT().Consume(matcher.Context, mock.AnythingOfType("*string"), map[string]string{
		stream.AttributeEncoding: stream.EncodingJson.String(),
		stream.AttributeRetry:    "true",
	}).RunAndReturn(func(ctx context.Context, _ any, _ map[string]string) (bool, error) {
		close(processingStarted)
		<-ctx.Done()
		close(processingCanceled)

		return true, nil
	}).Once()

	runDone := make(chan error, 1)
	go func() {
		runDone <- s.consumer.Run(s.kernelCtx)
	}()

	<-processingStarted
	s.clock.BlockUntilTimers(1)
	s.clock.Advance(time.Second)

	select {
	case <-processingCanceled:
	case <-time.After(time.Second):
		s.FailNow("retry processing was not canceled when the drain grace time expired")
	}

	select {
	case err := <-runDone:
		s.NoError(err)
	case <-time.After(time.Second):
		s.FailNow("consumer did not finish after retry processing was canceled")
	}
}

// TestRun_InputAndCallbackShareOneProcessingDeadline guards the single authoritative processing deadline. Inputs like
// kafka and kinesis keep records they handed out alive past their own cancellation and commit them afterwards, so they
// need to know when processing ends. If they timed that out on their own, the two deadlines would race over the same
// record: the input could stop trusting a result the callback still produces, or commit a record the callback was
// already denied the time to process.
func (s *ConsumerTestSuite) TestRun_InputAndCallbackShareOneProcessingDeadline() {
	s.settings.GraceTime = time.Second
	s.buildConsumer()

	msg := stream.NewJsonMessage(`"foo"`)
	processingStarted := make(chan struct{})
	processingCanceled := make(chan struct{})
	inputProcessingCanceled := make(chan struct{})
	drainAttached := make(chan struct{})

	s.expectRetryInputRun()
	s.input.EXPECT().
		Run(matcher.Context, mock.Anything).
		RunAndReturn(func(ctx context.Context, process stream.InputProcess) error {
			drainCtx, ok := exec.DrainContextFrom(ctx)
			close(drainAttached)
			if !s.True(ok, "the consumer must publish its processing deadline to the input") {
				return nil
			}

			// Mimic how kafka and kinesis hand out records: the record outlives the cancellation of the input
			// itself and is only abandoned once the shared deadline expires.
			inputCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
			defer cancel()
			stopDrainPropagation := context.AfterFunc(drainCtx, cancel)
			defer stopDrainPropagation()

			process(inputCtx, msg)

			// The callback and the input observe the same deadline, but through independent AfterFunc
			// registrations, so wait for it instead of assuming it already fired when the callback returned.
			<-inputCtx.Done()
			close(inputProcessingCanceled)

			return nil
		}).
		Once()

	s.callback.EXPECT().Run(matcher.Context).RunAndReturn(func(ctx context.Context) error {
		<-ctx.Done()

		return nil
	}).Once()
	s.callback.EXPECT().GetModel(mock.AnythingOfType("map[string]string")).Return(mdl.Box(""), nil).Once()
	s.callback.EXPECT().Consume(matcher.Context, mock.AnythingOfType("*string"), map[string]string{
		stream.AttributeEncoding: stream.EncodingJson.String(),
	}).RunAndReturn(func(ctx context.Context, _ any, _ map[string]string) (bool, error) {
		close(processingStarted)
		<-ctx.Done()
		close(processingCanceled)

		return true, nil
	}).Once()

	runDone := make(chan error, 1)
	go func() {
		runDone <- s.consumer.Run(s.kernelCtx)
	}()

	<-drainAttached

	select {
	case <-processingStarted:
	case <-time.After(time.Second):
		s.FailNow("message processing did not start")
	}

	// Stopping the input must not end processing: only the consumer's grace time may.
	s.kernelCancel()

	select {
	case <-processingCanceled:
		s.FailNow("processing was canceled before the consumer's grace time expired")
	case <-time.After(50 * time.Millisecond):
	}

	s.clock.BlockUntilTimers(1)
	s.clock.Advance(time.Second)

	select {
	case <-processingCanceled:
	case <-time.After(time.Second):
		s.FailNow("processing was not canceled when the consumer's grace time expired")
	}

	select {
	case <-inputProcessingCanceled:
	case <-time.After(time.Second):
		s.FailNow("the input did not observe the end of the processing deadline")
	}

	select {
	case err := <-runDone:
		s.NoError(err)
	case <-time.After(time.Second):
		s.FailNow("consumer did not finish after the processing deadline expired")
	}
}
