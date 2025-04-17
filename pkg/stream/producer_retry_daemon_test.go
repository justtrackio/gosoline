package stream_test

import (
	"context"
	"testing"
	"time"

	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/metric"
	"github.com/justtrackio/gosoline/pkg/stream"
	streamMocks "github.com/justtrackio/gosoline/pkg/stream/mocks"
	uuidMocks "github.com/justtrackio/gosoline/pkg/uuid/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestDaemon(input stream.Input, handler stream.RetryHandler, output stream.Output) stream.ProducerRetryDaemon {
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll)
	writer := metric.NewWriter()
	name := "test"
	uuidMock := &uuidMocks.Uuid{}
	uuidMock.EXPECT().NewV4().Return("uuid").Maybe()

	return stream.NewProducerRetryDaemonWithInterfaces(
		name,
		logger,
		writer,
		uuidMock,
		input,
		handler,
		output,
	)
}

func TestRetryOne_WithValidMessage(t *testing.T) {
	handler := new(streamMocks.RetryHandler)
	daemon := newTestDaemon(nil, handler, nil)

	msg := &stream.Message{
		Body: "body",
		Attributes: map[string]string{
			"goso.retry":    "true",
			"goso.retry.id": "uuid",
		},
	}

	handler.On("Put", mock.Anything, msg).Return(nil)

	err := daemon.RetryOne(context.Background(), msg)
	assert.NoError(t, err)
	handler.AssertCalled(t, "Put", mock.Anything, msg)
}

func TestRetryOne_WithInvalidMessage(t *testing.T) {
	handler := new(streamMocks.RetryHandler)
	daemon := newTestDaemon(nil, handler, nil)

	err := daemon.RetryOne(context.Background(), &stream.RawMessage{})
	assert.ErrorContains(t, err, "can not cast messages to message struct")
}

func TestRun_CancelsGracefully(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	input := new(streamMocks.Input)
	output := new(streamMocks.Output)

	input.On("Run", mock.Anything).Return(nil)
	input.On("Data").Return(make(<-chan *stream.Message))
	input.On("Stop").Return(nil)

	daemon := newTestDaemon(input, nil, output)
	go func() {
		time.Sleep(200 * time.Millisecond)
		input.Stop(ctx)
	}()

	err := daemon.Run(ctx)
	assert.NoError(t, err)
	input.AssertExpectations(t)
}

func TestIngestDataFromSource_AcknowledgeableInput(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	msg := &stream.Message{Body: "body"}

	input := new(streamMocks.AcknowledgeableInput)
	output := new(streamMocks.Output)

	// Simulate message data stream.
	ch := make(chan *stream.Message, 1)
	ch <- msg
	close(ch)

	input.On("Data").Return((<-chan *stream.Message)(ch))
	output.On("WriteOne", mock.Anything, mock.Anything).Return(nil)
	input.On("Ack", mock.Anything, msg, true).Return(nil)
	input.On("Run", mock.Anything).Return(nil)
	input.On("Stop").Return(nil)

	daemon := newTestDaemon(input, nil, output)
	err := daemon.Run(ctx) // IngestDataFromSource will be called during Run.
	assert.NoError(t, err)

	// Assert expectations for mocks
	input.AssertExpectations(t)
	output.AssertExpectations(t)
}
