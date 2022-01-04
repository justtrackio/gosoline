package stream_test

import (
	"context"
	"fmt"
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
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type BatchConsumerTestSuite struct {
	suite.Suite

	data    chan *stream.Message
	dataOut <-chan *stream.Message
	once    sync.Once
	stop    func()

	input *mocks.AcknowledgeableInput

	callback      *mocks.RunnableBatchConsumerCallback
	batchConsumer *stream.BatchConsumer
}

func TestBatchConsumerTestSuite(t *testing.T) {
	suite.Run(t, new(BatchConsumerTestSuite))
}

func (s *BatchConsumerTestSuite) SetupTest() {
	s.data = make(chan *stream.Message, 10)
	s.dataOut = s.data
	s.once = sync.Once{}
	s.stop = func() {
		s.once.Do(func() {
			close(s.data)
		})
	}

	s.input = new(mocks.AcknowledgeableInput)
	s.callback = new(mocks.RunnableBatchConsumerCallback)

	logger := logMocks.NewLoggerMockedAll()
	tracer := tracing.NewNoopTracer()
	mw := metricMocks.NewWriterMockedAll()
	me := stream.NewMessageEncoder(&stream.MessageEncoderSettings{})
	ticker := time.NewTicker(time.Second)
	settings := &stream.ConsumerSettings{
		Input:       "test",
		RunnerCount: 1,
		IdleTimeout: time.Second,
	}
	batchSettings := &stream.BatchConsumerSettings{
		IdleTimeout: time.Second,
		BatchSize:   5,
	}

	baseConsumer := stream.NewBaseConsumerWithInterfaces(logger, mw, tracer, s.input, me, s.callback, settings, "test", cfg.AppId{})
	s.batchConsumer = stream.NewBatchConsumerWithInterfaces(baseConsumer, s.callback, ticker, batchSettings)
}

func (s *BatchConsumerTestSuite) TestRun_ProcessOnStop() {
	s.input.
		On("Data").
		Return(s.dataOut)

	s.input.
		On("Run", mock.AnythingOfType("*context.cancelCtx")).
		Run(func(args mock.Arguments) {
			s.data <- stream.NewJsonMessage(`"foo"`)
			s.data <- stream.NewJsonMessage(`"bar"`)
			s.data <- stream.NewJsonMessage(`"foobar"`)
			s.stop()
		}).Return(nil)

	processed := 0

	s.input.
		On("Stop").
		Once()

	s.input.
		On("AckBatch", mock.AnythingOfType("*context.cancelCtx"), mock.AnythingOfType("[]*stream.Message")).
		Run(func(args mock.Arguments) {
			msgs := args[1].([]*stream.Message)
			processed = len(msgs)
		}).
		Return(nil).
		Once()

	acks := []bool{true, true, true}
	s.callback.On("Consume", mock.AnythingOfType("*context.cancelCtx"), mock.AnythingOfType("[]interface {}"), mock.AnythingOfType("[]map[string]interface {}")).
		Return(acks, nil)

	s.callback.On("GetModel", mock.AnythingOfType("map[string]interface {}")).
		Return(func(_ map[string]interface{}) interface{} {
			return mdl.String("")
		})

	s.callback.On("Run", mock.AnythingOfType("*context.cancelCtx")).
		Return(nil)

	err := s.batchConsumer.Run(context.Background())

	s.NoError(err, "there should be no error during run")
	s.Equal(3, processed)

	s.input.AssertExpectations(s.T())
	s.callback.AssertExpectations(s.T())
}

func (s *BatchConsumerTestSuite) TestRun_BatchSizeReached() {
	s.input.
		On("Data").
		Return(s.dataOut)

	s.input.
		On("Run", mock.AnythingOfType("*context.cancelCtx")).
		Run(func(args mock.Arguments) {
			s.data <- stream.NewJsonMessage(`"foo"`)
			s.data <- stream.NewJsonMessage(`"bar"`)
			s.data <- stream.NewJsonMessage(`"foobar"`)
			s.data <- stream.NewJsonMessage(`"barfoo"`)
			s.data <- stream.NewJsonMessage(`"foobarfoo"`)
		}).Return(nil)

	s.input.
		On("Stop").
		Once()

	processed := 0

	s.input.
		On("AckBatch", mock.AnythingOfType("*context.cancelCtx"), mock.AnythingOfType("[]*stream.Message")).
		Run(func(args mock.Arguments) {
			msgs := args[1].([]*stream.Message)
			processed = len(msgs)

			s.stop()
		}).
		Return(nil)

	acks := []bool{true, true, true, true, true}
	s.callback.On("Consume", mock.AnythingOfType("*context.cancelCtx"), mock.AnythingOfType("[]interface {}"), mock.AnythingOfType("[]map[string]interface {}")).
		Return(acks, nil)

	s.callback.On("GetModel", mock.AnythingOfType("map[string]interface {}")).
		Return(func(_ map[string]interface{}) interface{} {
			return mdl.String("")
		})

	s.callback.On("Run", mock.AnythingOfType("*context.cancelCtx")).
		Return(nil)

	err := s.batchConsumer.Run(context.Background())

	s.NoError(err, "there should be no error during run")
	s.Equal(5, processed)

	s.input.AssertExpectations(s.T())
	s.callback.AssertExpectations(s.T())
}

func (s *BatchConsumerTestSuite) TestRun_ContextCanceled() {
	ctx, cancel := context.WithCancel(context.Background())
	stopped := make(chan struct{})
	once := sync.Once{}

	s.input.
		On("Data").
		Return(s.dataOut)

	s.input.
		On("Run", mock.AnythingOfType("*context.cancelCtx")).
		Run(func(args mock.Arguments) {
			cancel()
			<-stopped
			s.stop()
		}).Return(nil)

	processed := 0

	s.input.
		On("Stop").
		Run(func(args mock.Arguments) {
			once.Do(func() {
				close(stopped)
			})
		}).
		Return(nil)

	s.callback.On("Run", mock.AnythingOfType("*context.cancelCtx")).
		Return(nil)

	err := s.batchConsumer.Run(ctx)

	s.NoError(err, "there should be no error during run")
	s.Equal(0, processed)

	s.input.AssertExpectations(s.T())
	s.callback.AssertExpectations(s.T())
}

func (s *BatchConsumerTestSuite) TestRun_InputRunError() {
	s.input.
		On("Data").
		Return(s.dataOut)
	s.input.
		On("Stop").
		Once()

	s.input.
		On("Run", mock.AnythingOfType("*context.cancelCtx")).
		Return(fmt.Errorf("read error"))

	s.callback.
		On("Run", mock.AnythingOfType("*context.cancelCtx")).
		Run(func(args mock.Arguments) {
			<-args[0].(context.Context).Done()
		}).Return(nil)

	err := s.batchConsumer.Run(context.Background())

	s.EqualError(err, "error while waiting for all routines to stop: panic during run of the consumer input: read error")

	s.input.AssertExpectations(s.T())
	s.callback.AssertExpectations(s.T())
}

func (s *BatchConsumerTestSuite) TestRun_CallbackRunError() {
	s.input.On("Data").
		Return(s.dataOut)
	s.input.On("Stop").
		Once()

	s.input.On("Run", mock.AnythingOfType("*context.cancelCtx")).
		Run(func(args mock.Arguments) {
			<-args[0].(context.Context).Done()
		}).
		Return(nil)

	s.callback.On("Run", mock.AnythingOfType("*context.cancelCtx")).
		Return(fmt.Errorf("consumerCallback run error"))

	err := s.batchConsumer.Run(context.Background())

	s.EqualError(err, "error while waiting for all routines to stop: panic during run of the consumerCallback: consumerCallback run error")

	s.input.AssertExpectations(s.T())
	s.callback.AssertExpectations(s.T())
}

func (s *BatchConsumerTestSuite) TestRun_AggregateMessage() {
	message1 := stream.NewJsonMessage(`"foo"`, map[string]interface{}{
		"attr1": "a",
	})
	message2 := stream.NewJsonMessage(`"bar"`, map[string]interface{}{
		"attr1": "b",
	})

	aggregateBody, err := json.Marshal([]stream.WritableMessage{message1, message2})
	s.Require().NoError(err)

	aggregate := stream.BuildAggregateMessage(string(aggregateBody))

	s.input.
		On("Data").
		Return(s.dataOut)

	s.input.
		On("Run", mock.AnythingOfType("*context.cancelCtx")).
		Run(func(args mock.Arguments) {
			s.data <- aggregate
			s.stop()
		}).Return(nil).
		Once()

	s.input.
		On("Stop").
		Once()

	processed := 0
	s.input.
		On("AckBatch", mock.AnythingOfType("*context.cancelCtx"), mock.AnythingOfType("[]*stream.Message")).
		Run(func(args mock.Arguments) {
			msgs := args[1].([]*stream.Message)
			processed = len(msgs)
		}).
		Return(nil).
		Once()

	s.callback.On("Run", mock.AnythingOfType("*context.cancelCtx")).
		Return(nil)

	s.callback.On("Consume", mock.AnythingOfType("*context.cancelCtx"), mock.AnythingOfType("[]interface {}"), mock.AnythingOfType("[]map[string]interface {}")).
		Return([]bool{true, true}, nil)

	s.callback.
		On("GetModel", mock.AnythingOfType("map[string]interface {}")).
		Return(mdl.String("")).
		Twice()

	err = s.batchConsumer.Run(context.Background())

	s.Nil(err, "there should be no error returned on consume")
	s.Equal(processed, 2)

	s.input.AssertExpectations(s.T())
	s.callback.AssertExpectations(s.T())
}
