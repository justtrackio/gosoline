package stream_test

import (
	"context"
	"fmt"
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
	"github.com/justtrackio/gosoline/pkg/stream/mocks"
	"github.com/justtrackio/gosoline/pkg/tracing"
	"github.com/justtrackio/gosoline/pkg/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

func TestBatchConsumerTestSuite(t *testing.T) {
	suite.Run(t, new(BatchConsumerTestSuite))
}

type BatchConsumerTestSuite struct {
	suite.Suite

	once         sync.Once
	kernelCtx    context.Context
	kernelCancel context.CancelFunc
	inputData    chan *stream.Message
	inputDataOut <-chan *stream.Message
	inputStop    func()

	input *mocks.AcknowledgeableInput

	callback      *mocks.RunnableBatchConsumerCallback
	batchConsumer *stream.BatchConsumer
}

func (s *BatchConsumerTestSuite) SetupTest() {
	s.once = sync.Once{}
	s.kernelCtx, s.kernelCancel = context.WithCancel(context.Background())

	s.inputData = make(chan *stream.Message, 10)
	s.inputDataOut = s.inputData
	s.inputStop = func() {
		s.once.Do(func() {
			close(s.inputData)
		})
	}

	s.input = mocks.NewAcknowledgeableInput(s.T())
	s.callback = mocks.NewRunnableBatchConsumerCallback(s.T())

	uuidGen := uuid.New()
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(s.T()))
	tracer := tracing.NewLocalTracer()
	mw := metricMocks.NewWriterMockedAll()
	me := stream.NewMessageEncoder(&stream.MessageEncoderSettings{})
	retryInput := stream.NewNoopInput()
	retryHandler := stream.NewRetryHandlerNoopWithInterfaces()

	ticker := clock.Provider.NewTicker(time.Second)
	settings := &stream.ConsumerSettings{
		Input:       "test",
		RunnerCount: 1,
		IdleTimeout: time.Second,
	}
	batchSettings := &stream.BatchConsumerSettings{
		IdleTimeout: time.Second,
		BatchSize:   5,
	}

	baseConsumer := stream.NewBaseConsumerWithInterfaces(
		uuidGen,
		logger,
		mw,
		tracer,
		s.input,
		me,
		retryInput,
		retryHandler,
		s.callback,
		settings,
		"test",
		cfg.AppId{},
	)
	s.batchConsumer = stream.NewBatchConsumerWithInterfaces(baseConsumer, s.callback, ticker, batchSettings)
}

func (s *BatchConsumerTestSuite) TestRun_ProcessOnStop() {
	s.input.EXPECT().Data().Return(s.inputDataOut)
	s.input.EXPECT().Stop().Run(s.inputStop).Once()

	s.input.
		EXPECT().
		Run(mock.AnythingOfType("*context.cancelCtx")).
		Run(func(ctx context.Context) {
			s.inputData <- stream.NewJsonMessage(`"foo"`)
			s.inputData <- stream.NewJsonMessage(`"bar"`)
			s.inputData <- stream.NewJsonMessage(`"foobar"`)
			s.kernelCancel()
		}).Return(nil)

	processed := 0
	acks := []bool{true, false, true}
	s.input.
		EXPECT().
		AckBatch(mock.AnythingOfType("*exec.stoppableContext"), mock.AnythingOfType("[]*stream.Message"), acks).
		Run(func(ctx context.Context, msgs []*stream.Message, acks []bool) {
			processed = len(msgs)
		}).
		Return(nil).
		Once()

	s.callback.EXPECT().
		Consume(mock.AnythingOfType("*context.valueCtx"), mock.AnythingOfType("[]interface {}"), mock.AnythingOfType("[]map[string]string")).
		Return(acks, nil).
		Once()

	s.callback.EXPECT().GetModel(mock.AnythingOfType("map[string]string")).
		Return(func(_ map[string]string) interface{} {
			return mdl.Box("")
		}).
		Times(3)

	s.callback.EXPECT().Run(mock.AnythingOfType("*context.cancelCtx")).
		Return(nil).
		Once()

	err := s.batchConsumer.Run(s.kernelCtx)

	s.NoError(err, "there should be no error during run")
	s.Equal(3, processed)
}

func (s *BatchConsumerTestSuite) TestRun_BatchSizeReached() {
	s.input.EXPECT().Data().Return(s.inputDataOut)
	s.input.EXPECT().Stop().Run(s.inputStop).Once()

	s.input.
		EXPECT().
		Run(mock.AnythingOfType("*context.cancelCtx")).
		Run(func(ctx context.Context) {
			s.inputData <- stream.NewJsonMessage(`"foo"`)
			s.inputData <- stream.NewJsonMessage(`"bar"`)
			s.inputData <- stream.NewJsonMessage(`"foobar"`)
			s.inputData <- stream.NewJsonMessage(`"barfoo"`)
			s.inputData <- stream.NewJsonMessage(`"foobarfoo"`)
		}).Return(nil).
		Once()

	processed := 0
	acks := []bool{true, false, true, false, true}
	s.input.
		EXPECT().
		AckBatch(mock.AnythingOfType("*exec.stoppableContext"), mock.AnythingOfType("[]*stream.Message"), acks).
		Run(func(ctx context.Context, msgs []*stream.Message, acks []bool) {
			processed = len(msgs)

			s.kernelCancel()
		}).
		Return(nil)

	s.callback.EXPECT().
		Consume(mock.AnythingOfType("*context.valueCtx"), mock.AnythingOfType("[]interface {}"), mock.AnythingOfType("[]map[string]string")).
		Return(acks, nil).
		Once()

	s.callback.EXPECT().GetModel(mock.AnythingOfType("map[string]string")).
		Return(func(_ map[string]string) interface{} {
			return mdl.Box("")
		}).
		Times(5)

	s.callback.EXPECT().Run(mock.AnythingOfType("*context.cancelCtx")).
		Return(nil).
		Once()

	err := s.batchConsumer.Run(s.kernelCtx)

	s.NoError(err, "there should be no error during run")
	s.Equal(5, processed)
}

func (s *BatchConsumerTestSuite) TestRun_InputRunError() {
	s.input.EXPECT().Data().Return(s.inputDataOut)
	s.input.EXPECT().Stop().Run(s.inputStop).Once()

	s.input.
		EXPECT().
		Run(mock.AnythingOfType("*context.cancelCtx")).
		Return(fmt.Errorf("read error")).
		Once()

	s.callback.
		EXPECT().
		Run(mock.AnythingOfType("*context.cancelCtx")).
		Run(func(ctx context.Context) {
			<-ctx.Done()
		}).
		Return(nil).
		Once()

	err := s.batchConsumer.Run(s.kernelCtx)

	s.EqualError(err, "error while waiting for all routines to stop: panic during run of the consumer input: read error")
}

func (s *BatchConsumerTestSuite) TestRun_CallbackRunError() {
	s.input.EXPECT().Data().Return(s.inputDataOut)
	s.input.EXPECT().Stop().Run(s.inputStop).Once()

	s.input.EXPECT().
		Run(mock.AnythingOfType("*context.cancelCtx")).
		Run(func(ctx context.Context) {
			<-ctx.Done()
		}).
		Return(nil).
		Once()

	s.callback.EXPECT().
		Run(mock.AnythingOfType("*context.cancelCtx")).
		Return(fmt.Errorf("consumerCallback run error")).
		Once()

	err := s.batchConsumer.Run(s.kernelCtx)

	s.EqualError(err, "error while waiting for all routines to stop: panic during run of the consumerCallback: consumerCallback run error")
}

func (s *BatchConsumerTestSuite) TestRun_AggregateMessage() {
	message1 := stream.NewJsonMessage(`"foo"`, map[string]string{
		"attr1": "a",
	})
	message2 := stream.NewJsonMessage(`"bar"`, map[string]string{
		"attr1": "b",
	})

	aggregateBody, err := json.Marshal([]stream.WritableMessage{message1, message2})
	s.Require().NoError(err)

	aggregate := stream.BuildAggregateMessage(string(aggregateBody))

	s.input.EXPECT().Data().Return(s.inputDataOut)
	s.input.EXPECT().Stop().Run(s.inputStop).Once()

	s.input.
		EXPECT().
		Run(mock.AnythingOfType("*context.cancelCtx")).
		Run(func(ctx context.Context) {
			s.inputData <- aggregate
			s.kernelCancel()
		}).
		Return(nil).
		Once()

	processed := 0
	acks := []bool{true, true}
	s.input.
		EXPECT().
		AckBatch(mock.AnythingOfType("*exec.stoppableContext"), mock.AnythingOfType("[]*stream.Message"), acks).
		Run(func(ctx context.Context, msgs []*stream.Message, acks []bool) {
			processed = len(msgs)
		}).
		Return(nil).
		Once()

	s.callback.EXPECT().
		Run(mock.AnythingOfType("*context.cancelCtx")).
		Return(nil).
		Once()

	s.callback.EXPECT().
		Consume(mock.AnythingOfType("*context.valueCtx"), mock.AnythingOfType("[]interface {}"), mock.AnythingOfType("[]map[string]string")).
		Return(acks, nil).
		Once()

	s.callback.
		EXPECT().
		GetModel(mock.AnythingOfType("map[string]string")).
		Return(mdl.Box("")).
		Twice()

	err = s.batchConsumer.Run(s.kernelCtx)

	s.Nil(err, "there should be no error returned on consume")
	s.Equal(processed, 2)
}
