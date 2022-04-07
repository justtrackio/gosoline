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

type ConsumerTestSuite struct {
	suite.Suite

	kernelCtx    context.Context
	kernelCancel context.CancelFunc

	input         *mocks.Input
	inputData     chan *stream.Message
	inputDataOut  <-chan *stream.Message
	inputStopOnce sync.Once
	inputStop     func(args mock.Arguments)

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

	s.input = new(mocks.Input)
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

	s.retryHandler = new(mocks.RetryHandler)
	s.retryHandler.On("Data").Return(s.retryDataOut)
	s.retryHandler.On("Stop").Run(s.retryStop).Once()

	s.uuidGen = new(uuidMocks.Uuid)
	s.callback = new(mocks.RunnableConsumerCallback)

	logger := logMocks.NewLoggerMockedAll()
	tracer := tracing.NewNoopTracer()
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

	baseConsumer := stream.NewBaseConsumerWithInterfaces(s.uuidGen, logger, mw, tracer, s.input, me, s.retryHandler, s.callback, settings, "test", cfg.AppId{})
	s.consumer = stream.NewConsumerWithInterfaces(baseConsumer, s.callback)
}

func (s *ConsumerTestSuite) TestGetModelNil() {
	s.retryHandler.On("Run", mock.AnythingOfType("*context.cancelCtx")).Return(nil)
	s.input.On("Run", mock.AnythingOfType("*context.cancelCtx")).Run(func(args mock.Arguments) {
		s.inputData <- stream.NewJsonMessage(`"foo"`, map[string]interface{}{
			"bla": "blub",
		})
		s.kernelCancel()
	}).Return(nil)

	s.callback.On("GetModel", mock.AnythingOfType("map[string]interface {}")).Return(func(_ map[string]interface{}) interface{} {
		return nil
	})
	s.callback.On("Run", mock.AnythingOfType("*context.cancelCtx")).Return(nil)

	err := s.consumer.Run(s.kernelCtx)

	s.NoError(err, "there should be no error during run")
	s.input.AssertExpectations(s.T())
	s.retryHandler.AssertExpectations(s.T())
	s.callback.AssertExpectations(s.T())
}

func (s *ConsumerTestSuite) TestRun() {
	s.retryHandler.On("Run", mock.AnythingOfType("*context.cancelCtx")).Return(nil)
	s.input.On("Run", mock.AnythingOfType("*context.cancelCtx")).Run(func(args mock.Arguments) {
		s.inputData <- stream.NewJsonMessage(`"foo"`)
		s.inputData <- stream.NewJsonMessage(`"bar"`)
		s.inputData <- stream.NewJsonMessage(`"foobar"`)
		s.kernelCancel()
	}).Return(nil)

	consumed := make([]*string, 0)
	s.callback.On("Consume", mock.AnythingOfType("*context.cancelCtx"), mock.AnythingOfType("*string"), map[string]interface{}{}).
		Run(func(args mock.Arguments) {
			consumed = append(consumed, args[1].(*string))
		}).Return(true, nil)

	s.callback.On("GetModel", mock.AnythingOfType("map[string]interface {}")).
		Return(func(_ map[string]interface{}) interface{} {
			return mdl.Box("")
		})

	s.callback.On("Run", mock.AnythingOfType("*context.cancelCtx")).Return(nil)

	err := s.consumer.Run(s.kernelCtx)

	s.NoError(err, "there should be no error during run")
	s.Len(consumed, 3)

	s.input.AssertExpectations(s.T())
	s.retryHandler.AssertExpectations(s.T())
	s.callback.AssertExpectations(s.T())
}

func (s *ConsumerTestSuite) TestRun_InputRunError() {
	s.retryHandler.On("Run", mock.AnythingOfType("*context.cancelCtx")).Return(nil)
	s.input.On("Run", mock.AnythingOfType("*context.cancelCtx")).Return(fmt.Errorf("read error"))

	s.callback.
		On("Run", mock.AnythingOfType("*context.cancelCtx")).
		Run(func(args mock.Arguments) {
			<-args[0].(context.Context).Done()
		}).Return(nil)

	err := s.consumer.Run(s.kernelCtx)

	s.EqualError(err, "error while waiting for all routines to stop: panic during run of the consumer input: read error")

	s.input.AssertExpectations(s.T())
	s.retryHandler.AssertExpectations(s.T())
	s.callback.AssertExpectations(s.T())
}

func (s *ConsumerTestSuite) TestRun_CallbackRunError() {
	s.retryHandler.On("Run", mock.AnythingOfType("*context.cancelCtx")).Return(nil)
	s.input.On("Run", mock.AnythingOfType("*context.cancelCtx")).
		Run(func(args mock.Arguments) {
			<-args[0].(context.Context).Done()
		}).
		Return(nil)

	s.callback.On("Run", mock.AnythingOfType("*context.cancelCtx")).
		Return(fmt.Errorf("consumerCallback run error"))

	err := s.consumer.Run(context.Background())

	s.EqualError(err, "error while waiting for all routines to stop: panic during run of the consumerCallback: consumerCallback run error")

	s.input.AssertExpectations(s.T())
	s.callback.AssertExpectations(s.T())
}

func (s *ConsumerTestSuite) TestRun_CallbackRunPanic() {
	consumed := make([]*string, 0)

	s.callback.
		On("Run", mock.AnythingOfType("*context.cancelCtx")).
		Return(nil)
	s.callback.
		On("Consume", mock.AnythingOfType("*context.cancelCtx"), mock.AnythingOfType("*string"), map[string]interface{}{}).
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
		On("GetModel", mock.AnythingOfType("map[string]interface {}")).
		Return(func(_ map[string]interface{}) interface{} {
			return mdl.Box("")
		})

	retryMsg := &stream.Message{
		Attributes: map[string]interface{}{
			stream.AttributeRetry:   true,
			stream.AttributeRetryId: "75828fe1-4c7d-4a21-99e5-03d63876ed23",
		},
		Body: `"foo"`,
	}

	s.uuidGen.
		On("NewV4").
		Return("75828fe1-4c7d-4a21-99e5-03d63876ed23")
	s.retryHandler.
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

	s.input.AssertExpectations(s.T())
	s.retryHandler.AssertExpectations(s.T())
	s.callback.AssertExpectations(s.T())
}

func (s *ConsumerTestSuite) TestRun_AggregateMessage() {
	s.retryHandler.On("Run", mock.AnythingOfType("*context.cancelCtx")).Return(nil)

	message1 := stream.NewJsonMessage(`"foo"`, map[string]interface{}{
		"attr1": "a",
	})
	message2 := stream.NewJsonMessage(`"bar"`, map[string]interface{}{
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

	expectedAttributes1 := map[string]interface{}{"attr1": "a"}
	s.callback.On("Consume", mock.AnythingOfType("*context.cancelCtx"), mock.AnythingOfType("*string"), expectedAttributes1).
		Run(func(args mock.Arguments) {
			ptr := args.Get(1).(*string)
			consumed = append(consumed, *ptr)
		}).
		Return(true, nil)

	expectedModelAttributes1 := map[string]interface{}{"attr1": "a", "encoding": "application/json"}
	s.callback.On("GetModel", expectedModelAttributes1).
		Return(mdl.Box(""))

	expectedAttributes2 := map[string]interface{}{"attr1": "b"}
	s.callback.On("Consume", mock.AnythingOfType("*context.cancelCtx"), mock.AnythingOfType("*string"), expectedAttributes2).
		Run(func(args mock.Arguments) {
			ptr := args.Get(1).(*string)
			consumed = append(consumed, *ptr)
		}).
		Return(true, nil)

	expectedModelAttributes2 := map[string]interface{}{"attr1": "b", "encoding": "application/json"}
	s.callback.On("GetModel", expectedModelAttributes2).
		Return(mdl.Box(""))

	err = s.consumer.Run(s.kernelCtx)

	s.Nil(err, "there should be no error returned on consume")
	s.Len(consumed, 2)
	s.Equal("foobar", strings.Join(consumed, ""))

	s.input.AssertExpectations(s.T())
	s.retryHandler.AssertExpectations(s.T())
	s.callback.AssertExpectations(s.T())
}

func (s *ConsumerTestSuite) TestRunWithRetry() {
	uuid := "243da976-c43f-4578-9307-596146e7dd9a"
	s.uuidGen.On("NewV4").Return(uuid)

	originalMessage := stream.NewJsonMessage(`"foo"`)
	retryMessage := stream.NewMessage(`"foo"`, map[string]interface{}{
		stream.AttributeRetry:   true,
		stream.AttributeRetryId: uuid,
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
	s.retryHandler.
		On("Run", mock.AnythingOfType("*context.cancelCtx")).
		Return(nil)

	consumed := make([]string, 0)
	s.callback.
		On("Consume", mock.AnythingOfType("*context.cancelCtx"), mock.AnythingOfType("*string"), map[string]interface{}{}).
		Run(func(args mock.Arguments) {
			consumed = append(consumed, *args[1].(*string))
		}).
		Return(false, nil).
		Once()
	s.callback.
		On("Consume", mock.AnythingOfType("*context.cancelCtx"), mock.AnythingOfType("*string"), map[string]interface{}{}).
		Run(func(args mock.Arguments) {
			consumed = append(consumed, *args[1].(*string))
			s.kernelCancel()
		}).
		Return(true, nil).
		Once()

	s.callback.
		On("GetModel", mock.AnythingOfType("map[string]interface {}")).
		Return(func(_ map[string]interface{}) interface{} {
			return mdl.Box("")
		}).
		Twice()

	s.callback.On("Run", mock.AnythingOfType("*context.cancelCtx")).Return(nil)

	err := s.consumer.Run(s.kernelCtx)

	s.NoError(err, "there should be no error during run")
	s.Equal("foo", consumed[0])
	s.Equal("foo from retry", consumed[1])

	s.input.AssertExpectations(s.T())
	s.retryHandler.AssertExpectations(s.T())
	s.callback.AssertExpectations(s.T())
}

func TestConsumerTestSuite(t *testing.T) {
	suite.Run(t, new(ConsumerTestSuite))
}
