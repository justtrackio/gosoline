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

func TestConsumer_Run(t *testing.T) {
	ch := make(chan *stream.Message)
	msg := &stream.Message{}
	ctx, closeFn := context.WithCancel(context.Background())

	logger := monMocks.NewLoggerMockedAll()
	span := new(tracingMocks.Span)
	tracer := new(tracingMocks.Tracer)
	input := new(streamMocks.AcknowledgeableInput)
	metricWriter := new(monMocks.MetricWriter)
	callback := new(streamMocks.ConsumerCallback)
	settings := stream.ConsumerSettings{
		RunnerCount: 1,
		IdleTimeout: 10 * time.Second, // long enough to cancel it before timeout
	}

	callback.On("Consume", ctx, msg).Run(func(_ mock.Arguments) {
		closeFn()
	}).Return(true, nil).Once()

	input.On("Data").Return(ch)
	input.On("Run", ctx).Return(nil)
	input.On("Ack", msg).Return(nil)
	input.On("Stop").Run(func(_ mock.Arguments) {
		close(ch)
	}).Return()

	tracer.On("StartSpanFromTraceAble", msg, "").Return(ctx, span)
	span.On("Finish").Return()

	metricWriter.On("WriteOne", mock.AnythingOfType("*mon.MetricDatum")).Return()

	consumer := stream.NewConsumerWithInterfaces(callback, logger, tracer, input, metricWriter, &settings)

	go func() {
		ch <- msg
	}()

	err := consumer.Run(ctx)

	assert.NoError(t, err)

	callback.AssertExpectations(t)
	input.AssertExpectations(t)
}
