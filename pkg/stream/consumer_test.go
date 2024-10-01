package stream_test

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/encoding/json"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/mdl"
	metricMocks "github.com/justtrackio/gosoline/pkg/metric/mocks"
	"github.com/justtrackio/gosoline/pkg/stream"
	"github.com/justtrackio/gosoline/pkg/stream/mocks"
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
	inputStop     func(args mock.Arguments)

	retryInput    *mocks.Input
	retryHandler  *mocks.RetryHandler
	retryData     chan *stream.Message
	retryDataOut  <-chan *stream.Message
	retryStopOnce sync.Once
	retryStop     func(args mock.Arguments)

	uuidGen  *uuidMocks.Uuid
	callback *mocks.RunnableConsumerCallback
	consumer *stream.Consumer
}

func (s *ConsumerTestSuite) SetupTest() {
	s.kernelCtx, s.kernelCancel = context.WithCancel(context.Background())

	s.inputData = make(chan *stream.Message, 10)
	s.inputDataOut = s.inputData
	s.inputStopOnce = sync.Once{}
	s.inputStop = func(args mock.Arguments) {
		s.inputStopOnce.Do(func() {
			close(s.inputData)
		})
	}

	s.input = mocks.NewAcknowledgeableInput(s.T())
	s.input.On("Data").Return(s.inputDataOut)
	s.input.On("Stop").Run(s.inputStop).Once()

	s.retryData = make(chan *stream.Message, 10)
	s.retryDataOut = s.retryData
	s.retryStopOnce = sync.Once{}
	s.retryStop = func(args mock.Arguments) {
		s.retryStopOnce.Do(func() {
			close(s.retryData)
		})
	}

	s.retryInput = mocks.NewInput(s.T())
	s.retryInput.On("Data").Return(s.retryDataOut)
	s.retryInput.On("Stop").Run(s.retryStop).Once()

	s.retryHandler = mocks.NewRetryHandler(s.T())

	s.uuidGen = uuidMocks.NewUuid(s.T())
	s.callback = mocks.NewRunnableConsumerCallback(s.T())

	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(s.T()))
	tracer := tracing.NewLocalTracer()
	mw := metricMocks.NewWriterMockedAll()
	me := stream.NewMessageEncoder(&stream.MessageEncoderSettings{})

	settings := &stream.ConsumerSettings{
		Input:       "test",
		RunnerCount: 1,
		IdleTimeout: time.Second,
		Retry: stream.ConsumerRetrySettings{
			Enabled: true,
		},
	}

	baseConsumer := stream.NewBaseConsumerWithInterfaces(s.uuidGen, logger, mw, tracer, s.input, me, s.retryInput, s.retryHandler, s.callback, settings, "test", cfg.AppId{})
	s.consumer = stream.NewConsumerWithInterfaces(baseConsumer, s.callback)
}

func (s *ConsumerTestSuite) TestGetModelNil() {
	s.retryInput.On("Run", mock.AnythingOfType("*context.cancelCtx")).Return(nil)
	s.input.On("Run", mock.AnythingOfType("*context.cancelCtx")).Run(func(args mock.Arguments) {
		s.inputData <- stream.NewJsonMessage(`"foo"`, map[string]string{
			"bla": "blub",
		})
		s.kernelCancel()
	}).Return(nil)

	s.input.
		On("Ack", mock.AnythingOfType("*context.cancelCtx"), mock.AnythingOfType("*stream.Message"), false).
		Return(nil).
		Once()
	s.callback.On("GetModel", mock.AnythingOfType("map[string]string")).Return(func(_ map[string]string) interface{} {
		return nil
	})
	s.callback.On("Run", mock.AnythingOfType("*context.cancelCtx")).Return(nil)

	err := s.consumer.Run(s.kernelCtx)

	s.NoError(err, "there should be no error during run")
}

func (s *ConsumerTestSuite) TestRun() {
	s.retryInput.On("Run", mock.AnythingOfType("*context.cancelCtx")).Return(nil)
	s.input.On("Run", mock.AnythingOfType("*context.cancelCtx")).Run(func(args mock.Arguments) {
		s.inputData <- stream.NewJsonMessage(`"foo"`)
		s.inputData <- stream.NewJsonMessage(`"bar"`)
		s.inputData <- stream.NewJsonMessage(`"foobar"`)
		s.kernelCancel()
	}).Return(nil)

	consumed := make([]*string, 0)

	s.input.
		On("Ack", mock.AnythingOfType("*context.cancelCtx"), mock.AnythingOfType("*stream.Message"), true).
		Return(nil).
		Times(3)

	s.callback.On("Consume", mock.AnythingOfType("*context.valueCtx"), mock.AnythingOfType("*string"), map[string]string{
		stream.AttributeEncoding: stream.EncodingJson.String(),
	}).
		Run(func(args mock.Arguments) {
			consumed = append(consumed, args[1].(*string))
		}).Return(true, nil)

	s.callback.On("GetModel", mock.AnythingOfType("map[string]string")).
		Return(func(_ map[string]string) interface{} {
			return mdl.Box("")
		})

	s.callback.On("Run", mock.AnythingOfType("*context.cancelCtx")).Return(nil)

	err := s.consumer.Run(s.kernelCtx)

	s.NoError(err, "there should be no error during run")
	s.Len(consumed, 3)
}

func (s *ConsumerTestSuite) TestRun_InputRunError() {
	s.retryInput.On("Run", mock.AnythingOfType("*context.cancelCtx")).Return(nil)
	s.input.On("Run", mock.AnythingOfType("*context.cancelCtx")).Return(fmt.Errorf("read error"))

	s.callback.
		On("Run", mock.AnythingOfType("*context.cancelCtx")).
		Run(func(args mock.Arguments) {
			<-args[0].(context.Context).Done()
		}).Return(nil)

	err := s.consumer.Run(s.kernelCtx)

	s.EqualError(err, "error while waiting for all routines to stop: panic during run of the consumer input: read error")
}

func (s *ConsumerTestSuite) TestRun_CallbackRunError() {
	s.retryInput.On("Run", mock.AnythingOfType("*context.cancelCtx")).Return(nil)
	s.input.On("Run", mock.AnythingOfType("*context.cancelCtx")).
		Run(func(args mock.Arguments) {
			<-args[0].(context.Context).Done()
		}).
		Return(nil)

	s.callback.On("Run", mock.AnythingOfType("*context.cancelCtx")).
		Return(fmt.Errorf("consumerCallback run error"))

	err := s.consumer.Run(context.Background())

	s.EqualError(err, "error while waiting for all routines to stop: panic during run of the consumerCallback: consumerCallback run error")
}

func (s *ConsumerTestSuite) TestRun_CallbackRunPanic() {
	consumed := make([]*string, 0)

	s.callback.
		On("Run", mock.AnythingOfType("*context.cancelCtx")).
		Return(nil)

	// 1 message should be acked.
	s.input.
		On("Ack", mock.AnythingOfType("*context.cancelCtx"), mock.AnythingOfType("*stream.Message"), true).
		Return(nil).
		Once()
	// 1 message should be nacked due to panic.
	s.input.
		On("Ack", mock.AnythingOfType("*context.cancelCtx"), mock.AnythingOfType("*stream.Message"), false).
		Return(nil).
		Once()

	s.callback.
		On("Consume", mock.AnythingOfType("*context.valueCtx"), mock.AnythingOfType("*string"), map[string]string{
			stream.AttributeEncoding: stream.EncodingJson.String(),
		}).
		Run(func(args mock.Arguments) {
			ptr := args.Get(1).(*string)
			consumed = append(consumed, ptr)

			msg := *ptr
			if msg == "foo" {
				panic("foo")
			}
		}).
		Return(true, nil)
	s.callback.
		On("GetModel", mock.AnythingOfType("map[string]string")).
		Return(func(_ map[string]string) interface{} {
			return mdl.Box("")
		})

	retryMsg := &stream.Message{
		Attributes: map[string]string{
			stream.AttributeEncoding: stream.EncodingJson.String(),
			stream.AttributeRetry:    "true",
			stream.AttributeRetryId:  "75828fe1-4c7d-4a21-99e5-03d63876ed23",
		},
		Body: `"foo"`,
	}

	s.uuidGen.
		On("NewV4").
		Return("75828fe1-4c7d-4a21-99e5-03d63876ed23")
	s.retryInput.
		On("Run", mock.AnythingOfType("*context.cancelCtx")).
		Return(nil)
	s.retryHandler.
		On("Put", mock.AnythingOfType("*context.valueCtx"), retryMsg).
		Return(nil)
	s.input.
		On("Run", mock.AnythingOfType("*context.cancelCtx")).
		Run(func(args mock.Arguments) {
			s.inputData <- stream.NewJsonMessage(`"foo"`)
			s.inputData <- stream.NewJsonMessage(`"bar"`)
			s.kernelCancel()
		}).
		Return(nil)

	err := s.consumer.Run(s.kernelCtx)

	s.Nil(err, "there should be no error returned on consume")
	s.Len(consumed, 2)
}

func (s *ConsumerTestSuite) TestRun_AggregateMessage() {
	s.retryInput.On("Run", mock.AnythingOfType("*context.cancelCtx")).Return(nil)

	message1 := stream.NewJsonMessage(`"foo"`, map[string]string{
		"attr1": "a",
	})
	message2 := stream.NewJsonMessage(`"bar"`, map[string]string{
		"attr1": "b",
	})

	aggregateBody, err := json.Marshal([]stream.WritableMessage{message1, message2})
	s.Require().NoError(err)

	aggregate := stream.BuildAggregateMessage(string(aggregateBody))

	s.input.On("Run", mock.AnythingOfType("*context.cancelCtx")).Run(func(args mock.Arguments) {
		s.inputData <- aggregate
		s.kernelCancel()
	}).Return(nil)

	consumed := make([]string, 0)
	s.callback.On("Run", mock.AnythingOfType("*context.cancelCtx")).Return(nil)

	expectedAttributes1 := map[string]string{
		stream.AttributeEncoding: stream.EncodingJson.String(),
		"attr1":                  "a",
	}

	s.input.
		On("Ack", mock.AnythingOfType("*context.cancelCtx"), mock.AnythingOfType("*stream.Message"), true).
		Return(nil).
		Once()
	s.callback.On("Consume", mock.AnythingOfType("*context.valueCtx"), mock.AnythingOfType("*string"), expectedAttributes1).
		Run(func(args mock.Arguments) {
			ptr := args.Get(1).(*string)
			consumed = append(consumed, *ptr)
		}).
		Return(true, nil)

	expectedModelAttributes1 := map[string]string{"attr1": "a", "encoding": "application/json"}
	s.callback.On("GetModel", expectedModelAttributes1).
		Return(mdl.Box(""))

	expectedAttributes2 := map[string]string{
		stream.AttributeEncoding: stream.EncodingJson.String(),
		"attr1":                  "b",
	}
	s.callback.On("Consume", mock.AnythingOfType("*context.valueCtx"), mock.AnythingOfType("*string"), expectedAttributes2).
		Run(func(args mock.Arguments) {
			ptr := args.Get(1).(*string)
			consumed = append(consumed, *ptr)
		}).
		Return(true, nil)

	expectedModelAttributes2 := map[string]string{"attr1": "b", "encoding": "application/json"}
	s.callback.On("GetModel", expectedModelAttributes2).
		Return(mdl.Box(""))

	err = s.consumer.Run(s.kernelCtx)

	s.Nil(err, "there should be no error returned on consume")
	s.Len(consumed, 2)
	s.Equal("foobar", strings.Join(consumed, ""))
}

func (s *ConsumerTestSuite) TestRunWithRetry() {
	uuid := "243da976-c43f-4578-9307-596146e7dd9a"
	s.uuidGen.On("NewV4").Return(uuid)

	originalMessage := stream.NewJsonMessage(`"foo"`)
	retryMessage := stream.NewMessage(`"foo"`, map[string]string{
		stream.AttributeEncoding: stream.EncodingJson.String(),
		stream.AttributeRetry:    "true",
		stream.AttributeRetryId:  uuid,
	})

	s.input.On("Run", mock.AnythingOfType("*context.cancelCtx")).Run(func(args mock.Arguments) {
		s.inputData <- originalMessage
	}).Return(nil)

	s.retryHandler.
		On("Put", mock.AnythingOfType("*context.valueCtx"), retryMessage).
		Run(func(args mock.Arguments) {
			s.retryData <- stream.NewJsonMessage(`"foo from retry"`)
		}).
		Return(nil).
		Once()
	s.retryInput.
		On("Run", mock.AnythingOfType("*context.cancelCtx")).
		Return(nil)

	consumed := make([]string, 0)

	// If a single sub-message in an aggregate fails then aggregate should be nacked.
	s.input.
		On("Ack", mock.AnythingOfType("*context.cancelCtx"), mock.AnythingOfType("*stream.Message"), false).
		Return(nil).
		Once()

	s.callback.
		On("Consume", mock.AnythingOfType("*context.valueCtx"), mock.AnythingOfType("*string"), map[string]string{
			stream.AttributeEncoding: stream.EncodingJson.String(),
		}).
		Run(func(args mock.Arguments) {
			consumed = append(consumed, *args[1].(*string))
		}).
		Return(false, nil).
		Once()
	s.callback.
		On("Consume", mock.AnythingOfType("*context.valueCtx"), mock.AnythingOfType("*string"), map[string]string{
			stream.AttributeEncoding: stream.EncodingJson.String(),
		}).
		Run(func(args mock.Arguments) {
			consumed = append(consumed, *args[1].(*string))
			s.kernelCancel()
		}).
		Return(true, nil).
		Once()

	s.callback.
		On("GetModel", mock.AnythingOfType("map[string]string")).
		Return(func(_ map[string]string) interface{} {
			return mdl.Box("")
		}).
		Twice()

	s.callback.On("Run", mock.AnythingOfType("*context.cancelCtx")).Return(nil)

	err := s.consumer.Run(s.kernelCtx)

	s.NoError(err, "there should be no error during run")
	s.Equal("foo", consumed[0])
	s.Equal("foo from retry", consumed[1])
}
