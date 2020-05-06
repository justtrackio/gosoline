package stream_test

import (
	"context"
	"fmt"
	monMocks "github.com/applike/gosoline/pkg/mon/mocks"
	"github.com/applike/gosoline/pkg/stream"
	"github.com/applike/gosoline/pkg/stream/mocks"
	"github.com/applike/gosoline/pkg/tracing"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"sync"
	"testing"
	"time"
)

type ConsumerTestSuite struct {
	suite.Suite

	data chan *stream.Message
	once sync.Once
	stop func()

	input *mocks.Input

	callback *mocks.FullConsumerCallback
	consumer *stream.Consumer
}

func (s *ConsumerTestSuite) SetupTest() {
	s.data = make(chan *stream.Message, 10)
	s.once = sync.Once{}
	s.stop = func() {
		s.once.Do(func() {
			close(s.data)
		})
	}

	s.input = new(mocks.Input)
	s.callback = new(mocks.FullConsumerCallback)

	logger := monMocks.NewLoggerMockedAll()
	tracer := tracing.NewNoopTracer()
	mw := monMocks.NewMetricWriterMockedAll()
	me := stream.NewMessageEncoder(&stream.MessageEncoderSettings{})

	s.consumer = stream.NewConsumer("test", s.callback)
	s.consumer.BootWithInterfaces(logger, tracer, mw, s.input, me, &stream.ConsumerSettings{
		Input:       "test",
		RunnerCount: 1,
		IdleTimeout: time.Second,
	})
}

func (s *ConsumerTestSuite) TestRun() {
	s.input.On("Data").Return(s.data)
	s.input.On("Run", mock.AnythingOfType("*context.cancelCtx")).Run(func(args mock.Arguments) {
		s.data <- stream.NewJsonMessage(`"foo"`)
		s.data <- stream.NewJsonMessage(`"bar"`)
		s.data <- stream.NewJsonMessage(`"foobar"`)
		s.stop()
	}).Return(nil)
	s.input.On("Stop")

	consumed := make([]*string, 0)
	s.callback.On("Consume", mock.AnythingOfType("*context.emptyCtx"), mock.AnythingOfType("*string"), map[string]interface{}{}).Run(func(args mock.Arguments) {
		consumed = append(consumed, args[1].(*string))
	}).Return(true, nil)
	s.callback.On("GetModel").Return(func() interface{} {
		model := ""
		return &model
	})
	s.callback.On("Run", mock.AnythingOfType("*context.cancelCtx")).Return(nil)

	err := s.consumer.Run(context.Background())

	s.NoError(err, "there should be no error during run")
	s.Len(consumed, 3)
	s.input.AssertExpectations(s.T())
}

func (s *ConsumerTestSuite) TestContextCancel() {
	ctx, cancel := context.WithCancel(context.Background())
	stopped := make(chan struct{})
	once := sync.Once{}

	s.input.On("Data").Return(s.data)
	s.input.On("Run", mock.AnythingOfType("*context.cancelCtx")).Run(func(args mock.Arguments) {
		cancel()
		<-stopped
		s.stop()
	}).Return(nil)
	s.input.On("Stop").Run(func(args mock.Arguments) {
		once.Do(func() {
			close(stopped)
		})
	})

	s.callback.On("Run", mock.AnythingOfType("*context.cancelCtx")).Return(nil)

	err := s.consumer.Run(ctx)

	s.NoError(err, "there should be no error during run")
}

func (s *ConsumerTestSuite) TestInputRunError() {
	s.input.On("Data").Return(s.data)
	s.input.On("Run", mock.AnythingOfType("*context.cancelCtx")).Return(fmt.Errorf("read error"))
	s.callback.On("Run", mock.AnythingOfType("*context.cancelCtx")).Run(func(args mock.Arguments) {
		<-args[0].(context.Context).Done()
	}).Return(nil)

	err := s.consumer.Run(context.Background())

	s.EqualError(err, "error while waiting for all routines to stop: read error")
	s.input.AssertExpectations(s.T())
}

func (s *ConsumerTestSuite) TestCallbackRunError() {
	s.input.On("Data").Return(s.data)
	s.input.On("Run", mock.AnythingOfType("*context.cancelCtx")).Run(func(args mock.Arguments) {
		<-args[0].(context.Context).Done()
	}).Return(nil)
	s.callback.On("Run", mock.AnythingOfType("*context.cancelCtx")).Return(fmt.Errorf("callback run error"))

	err := s.consumer.Run(context.Background())

	s.EqualError(err, "error while waiting for all routines to stop: callback run error")
	s.input.AssertExpectations(s.T())
}

func (s *ConsumerTestSuite) TestCallbackRunPanic() {
	s.input.On("Data").Return(s.data)
	s.input.On("Run", mock.AnythingOfType("*context.cancelCtx")).Run(func(args mock.Arguments) {
		s.data <- stream.NewJsonMessage(`"foo"`)
		s.data <- stream.NewJsonMessage(`"bar"`)
		s.stop()
	}).Return(nil)
	s.input.On("Stop")

	consumed := make([]*string, 0)

	s.callback.On("Run", mock.AnythingOfType("*context.cancelCtx")).Return(nil)
	s.callback.On("Consume", mock.AnythingOfType("*context.emptyCtx"), mock.AnythingOfType("*string"), map[string]interface{}{}).Run(func(args mock.Arguments) {
		ptr := args.Get(1).(*string)
		consumed = append(consumed, ptr)

		msg := *ptr
		if msg == "foo" {
			panic("foo")
		}
	}).Return(true, nil)
	s.callback.On("GetModel").Return(func() interface{} {
		model := ""
		return &model
	})

	err := s.consumer.Run(context.Background())

	s.Nil(err, "there should be no error returned on consume")
	s.Len(consumed, 2)
	s.input.AssertExpectations(s.T())
	s.callback.AssertExpectations(s.T())
}

func TestConsumerTestSuite(t *testing.T) {
	suite.Run(t, new(ConsumerTestSuite))
}
