package stream_test

import (
	"context"
	monMocks "github.com/applike/gosoline/pkg/mon/mocks"
	"github.com/applike/gosoline/pkg/stream"
	streamMocks "github.com/applike/gosoline/pkg/stream/mocks"
	tracingMocks "github.com/applike/gosoline/pkg/tracing/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
	"time"
)

func TestBatchConsumer_Run(t *testing.T) {
	msg := &stream.Message{}
	msgs := []*stream.Message{msg}
	ch := make(chan *stream.Message)
	ctx, closeFn := context.WithCancel(context.Background())

	logger := monMocks.NewLoggerMockedAll()
	span := new(tracingMocks.Span)
	tracer := new(tracingMocks.Tracer)
	input := new(streamMocks.AcknowledgeableInput)
	metricWriter := new(monMocks.MetricWriter)
	callback := new(streamMocks.BatchConsumerCallback)
	settings := stream.BatchConsumerSettings{
		BatchSize:   10,               // large enough not to be matched by the given messages
		IdleTimeout: 10 * time.Second, // long enough to cancel it before timeout
	}

	callback.On("Process", mock.AnythingOfType("*context.emptyCtx"), msgs).Return(msgs, nil).Once()

	input.On("Data").Return(ch)
	input.On("Run", ctx).Return(nil)
	input.On("AckBatch", msgs).Return(nil)
	input.On("Stop").Run(func(_ mock.Arguments) {
		close(ch)
	}).Return()

	tracer.On("StartSpanFromTraceAble", msgs, "").Return(ctx, span)
	span.On("Finish").Return()

	metricWriter.On("WriteOne", mock.AnythingOfType("*mon.MetricDatum")).Return()

	consumer := stream.NewBatchConsumerWithInterfaces(callback, logger, tracer, input, metricWriter, &settings)

	go func() {
		ch <- msg
		closeFn()
	}()

	err := consumer.Run(ctx)

	assert.NoError(t, err)

	callback.AssertExpectations(t)
	input.AssertExpectations(t)
}
