package stream_test

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/encoding/json"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/mdl"
	metricMocks "github.com/justtrackio/gosoline/pkg/metric/mocks"
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

	input         *mocks.AcknowledgeableInput
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
}

func (s *ConsumerTestSuite) SetupTest() {
	s.kernelCtx, s.kernelCancel = context.WithCancel(s.T().Context())

	s.inputData = make(chan *stream.Message, 10)
	s.inputDataOut = s.inputData
	s.inputStopOnce = sync.Once{}
	s.inputStop = func(context.Context) {
		s.inputStopOnce.Do(func() {
			close(s.inputData)
		})
	}

	s.input = mocks.NewAcknowledgeableInput(s.T())
	s.input.EXPECT().Data().Return(s.inputDataOut)
	s.input.EXPECT().Stop(matcher.Context).Run(s.inputStop).Once()

	s.retryData = make(chan *stream.Message, 10)
	s.retryDataOut = s.retryData
	s.retryStopOnce = sync.Once{}
	s.retryStop = func(context.Context) {
		s.retryStopOnce.Do(func() {
			close(s.retryData)
		})
	}

	s.retryInput = mocks.NewInput(s.T())
	s.retryInput.EXPECT().Data().Return(s.retryDataOut)
	s.retryInput.EXPECT().Stop(matcher.Context).Run(s.retryStop).Once()

	s.retryHandler = mocks.NewRetryHandler(s.T())

	s.uuidGen = uuidMocks.NewUuid(s.T())
	s.callback = mocks.NewRunnableUntypedConsumerCallback(s.T())

	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(s.T()))
	tracer := tracing.NewLocalTracer()
	mw := metricMocks.NewWriter(s.T())
	mw.EXPECT().Write(matcher.Context, mock.Anything).Return().Maybe()
	me := stream.NewMessageEncoder(&stream.MessageEncoderSettings{})

	settings := stream.ConsumerSettings{
		Input:       "test",
		RunnerCount: 1,
		IdleTimeout: time.Second,
		Retry: stream.ConsumerRetrySettings{
			Enabled: true,
		},
		Healthcheck: health.HealthCheckSettings{
			Timeout: time.Minute,
		},
		AggregateMessageMode: stream.AggregateMessageModeAtMostOnce,
	}

	healthCheckTimer := clock.NewHealthCheckTimerWithInterfaces(clock.NewFakeClock(), settings.Healthcheck.Timeout)

	baseConsumer := stream.NewBaseConsumerWithInterfaces(
		s.uuidGen,
		logger,
		mw,
		tracer,
		s.input,
		me,
		s.retryInput,
		s.retryHandler,
		s.callback,
		settings,
		"test",
		cfg.AppId{},
	)
	s.consumer = stream.NewUntypedConsumerWithInterfaces(baseConsumer, s.callback, healthCheckTimer)
}

func (s *ConsumerTestSuite) TestGetModelNil() {
	s.retryInput.EXPECT().Run(matcher.Context).Return(nil).Once()
	s.input.EXPECT().
		Run(matcher.Context).
		Run(func(ctx context.Context) {
			s.inputData <- stream.NewJsonMessage(`"foo"`, map[string]string{
				"bla": "blub",
			})
		}).
		Return(nil).
		Once()

	s.input.EXPECT().
		Ack(matcher.Context, mock.AnythingOfType("*stream.Message"), false).
		Run(func(ctx context.Context, msg *stream.Message, ack bool) {
			s.kernelCancel()
		}).
		Return(nil).
		Once()
	s.callback.EXPECT().GetModel(mock.AnythingOfType("map[string]string")).Return(func(_ map[string]string) any {
		return nil
	}).Once()
	s.callback.EXPECT().Run(matcher.Context).Return(nil).Once()

	err := s.consumer.Run(s.kernelCtx)

	s.NoError(err, "there should be no error during run")
}

func (s *ConsumerTestSuite) TestRun() {
	s.retryInput.EXPECT().Run(matcher.Context).Return(nil).Once()
	s.input.EXPECT().
		Run(matcher.Context).
		Run(func(ctx context.Context) {
			s.inputData <- stream.NewJsonMessage(`"foo"`)
			s.inputData <- stream.NewJsonMessage(`"bar"`)
			s.inputData <- stream.NewJsonMessage(`"foobar"`)
		}).
		Return(nil).
		Once()

	consumed := make([]*string, 0)

	s.input.EXPECT().
		Ack(matcher.Context, mock.AnythingOfType("*stream.Message"), true).
		Return(nil).
		Times(3)

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
		Return(func(_ map[string]string) any {
			return mdl.Box("")
		}).
		Times(3)

	s.callback.EXPECT().Run(matcher.Context).Return(nil).Once()

	err := s.consumer.Run(s.kernelCtx)

	s.NoError(err, "there should be no error during run")
	s.Len(consumed, 3)
}

func (s *ConsumerTestSuite) TestRun_InputRunError() {
	s.retryInput.EXPECT().Run(matcher.Context).Return(nil).Once()
	s.input.EXPECT().Run(matcher.Context).Return(fmt.Errorf("read error")).Once()

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
	s.retryInput.EXPECT().Run(matcher.Context).Return(nil).Once()
	s.input.EXPECT().
		Run(matcher.Context).
		Run(func(ctx context.Context) {
			<-ctx.Done()
		}).
		Return(nil).
		Once()

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

	// 1 message should be acked.
	s.input.EXPECT().
		Ack(matcher.Context, mock.AnythingOfType("*stream.Message"), true).
		Return(nil).
		Once()
	// 1 message should be nacked due to panic.
	s.input.EXPECT().
		Ack(matcher.Context, mock.AnythingOfType("*stream.Message"), false).
		Return(nil).
		Once()

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
		Return(func(_ map[string]string) any {
			return mdl.Box("")
		}).
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
	s.retryInput.EXPECT().Run(matcher.Context).Return(nil).Once()
	s.retryHandler.EXPECT().Put(matcher.Context, retryMsg).Return(nil).Once()
	s.input.EXPECT().
		Run(matcher.Context).
		Run(func(ctx context.Context) {
			s.inputData <- stream.NewJsonMessage(`"foo"`)
			s.inputData <- stream.NewJsonMessage(`"bar"`)
		}).
		Return(nil).
		Once()

	err := s.consumer.Run(s.kernelCtx)

	s.Nil(err, "there should be no error returned on consume")
	s.Len(consumed, 2)
}

func (s *ConsumerTestSuite) TestRun_AggregateMessage() {
	s.retryInput.EXPECT().Run(matcher.Context).Return(nil).Once()

	message1 := stream.NewJsonMessage(`"foo"`, map[string]string{
		"attr1": "a",
	})
	message2 := stream.NewJsonMessage(`"bar"`, map[string]string{
		"attr1": "b",
	})

	aggregateBody, err := json.Marshal([]stream.WritableMessage{message1, message2})
	s.NoError(err)

	aggregate := stream.BuildAggregateMessage(string(aggregateBody))

	s.input.EXPECT().
		Run(matcher.Context).
		Run(func(ctx context.Context) {
			s.inputData <- aggregate
		}).
		Return(nil).
		Once()

	consumed := make([]string, 0)
	s.callback.EXPECT().Run(matcher.Context).Return(nil).Once()

	expectedAttributes1 := map[string]string{
		stream.AttributeEncoding: stream.EncodingJson.String(),
		"attr1":                  "a",
	}

	s.input.EXPECT().
		Ack(matcher.Context, mock.AnythingOfType("*stream.Message"), true).
		Return(nil).
		Once()
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
		Return(mdl.Box("")).
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
		Return(mdl.Box("")).
		Once()

	err = s.consumer.Run(s.kernelCtx)

	s.Nil(err, "there should be no error returned on consume")
	s.Len(consumed, 2)
	s.Equal("foobar", strings.Join(consumed, ""))
}

func (s *ConsumerTestSuite) TestRunWithRetry() {
	uuid := "243da976-c43f-4578-9307-596146e7dd9a"
	s.uuidGen.EXPECT().NewV4().Return(uuid)

	originalMessage := stream.NewJsonMessage(`"foo"`)
	retryMessage := stream.NewMessage(`"foo"`, map[string]string{
		stream.AttributeEncoding: stream.EncodingJson.String(),
		stream.AttributeRetry:    "true",
		stream.AttributeRetryId:  uuid,
	})

	s.input.EXPECT().
		Run(matcher.Context).
		Run(func(ctx context.Context) {
			s.inputData <- originalMessage
		}).
		Return(nil).
		Once()

	s.retryHandler.EXPECT().
		Put(matcher.Context, retryMessage).
		Run(func(ctx context.Context, msg *stream.Message) {
			s.retryData <- stream.NewJsonMessage(`"foo from retry"`)
		}).
		Return(nil).
		Once()
	s.retryInput.EXPECT().Run(matcher.Context).Return(nil).Once()

	consumed := make([]string, 0)

	// If a single sub-message in an aggregate fails then aggregate should be nacked.
	s.input.EXPECT().
		Ack(matcher.Context, mock.AnythingOfType("*stream.Message"), false).
		Return(nil).
		Once()

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
		}).
		Run(func(ctx context.Context, model any, attributes map[string]string) {
			consumed = append(consumed, *model.(*string))
			s.kernelCancel()
		}).
		Return(true, nil).
		Once()

	s.callback.EXPECT().
		GetModel(mock.AnythingOfType("map[string]string")).
		Return(func(_ map[string]string) any {
			return mdl.Box("")
		}).
		Twice()

	s.callback.EXPECT().Run(matcher.Context).Return(nil)

	err := s.consumer.Run(s.kernelCtx)

	s.NoError(err, "there should be no error during run")
	s.Equal("foo", consumed[0])
	s.Equal("foo from retry", consumed[1])
}
