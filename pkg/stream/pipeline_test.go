package stream_test

import (
	"context"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/mon/mocks"
	"github.com/applike/gosoline/pkg/stream"
	streamMocks "github.com/applike/gosoline/pkg/stream/mocks"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

type callback struct {
}

func (c *callback) Boot(_ cfg.Config, _ mon.Logger) error {
	return nil
}

func (c *callback) Process(_ context.Context, messages []stream.PipelineInput) ([]stream.PipelineOutput, error) {
	output := make([]stream.PipelineOutput, len(messages))

	for i, input := range messages {
		for _, msg := range input.Messages {
			msg.Body = "foobaz"
		}
		output[i] = input.CreateOutput(input.Messages)
	}

	return output, nil
}

func runPipelineWithSettings(t *testing.T, settings *stream.PipelineSettings, input stream.Input, expectedMessages int, ctx context.Context) {
	logger := mocks.NewLoggerMockedAll()
	metric := mocks.NewMetricWriterMockedAll()
	output := stream.NewInMemoryOutput()

	callback := &callback{}
	pipe := stream.NewPipeline(callback)

	err := pipe.BootWithInterfaces(logger, metric, input, output, settings)
	assert.NoError(t, err, "the pipeline should boot without an error")

	err = pipe.Run(ctx)
	assert.NoError(t, err, "the pipeline should run without an error")

	size := output.Size()
	assert.Equal(t, expectedMessages, size, "the output should contain %d message(s)", expectedMessages)

	contains := output.ContainsBody("foobaz")
	assert.True(t, contains, "the output should contain the body 'foobaz'")
}

func TestPipeline_RunBatchSize(t *testing.T) {
	input := stream.NewInMemoryInput(&stream.InMemorySettings{Size: 1})
	input.Publish(&stream.Message{
		Body: "foobar",
	})
	input.Stop()

	runPipelineWithSettings(t, &stream.PipelineSettings{
		Interval:  time.Hour,
		BatchSize: 1,
	}, input, 1, context.Background())
}

func TestPipeline_RunAggregate(t *testing.T) {
	input := new(streamMocks.AcknowledgeableInput)

	aggregateMessage, err := stream.MarshalJsonMessage([]stream.Message{
		{
			Body: "a",
		},
		{
			Body: "b",
		},
		{
			Body: "c",
		},
		{
			Body: "d",
		},
	}, map[string]interface{}{
		stream.AttributeAggregate:        true,
		stream.AttributeSqsReceiptHandle: "receipt-1",
	})
	assert.NoError(t, err)

	dataChan := make(chan *stream.Message, 3)
	dataChan <- aggregateMessage
	dataChan <- &stream.Message{
		Attributes: map[string]interface{}{
			stream.AttributeSqsReceiptHandle: "receipt-2",
		},
		Body: "foobar",
	}
	dataChan <- &stream.Message{
		Attributes: map[string]interface{}{
			stream.AttributeSqsReceiptHandle: "receipt-3",
		},
		Body: "abc",
	}
	close(dataChan)

	input.On("AckBatch", []*stream.Message{
		{
			Attributes: map[string]interface{}{
				stream.AttributeAggregate:        true,
				stream.AttributeSqsReceiptHandle: "receipt-1",
			},
		},
	}).Return(nil).Once()
	input.On("AckBatch", []*stream.Message{
		{
			Attributes: map[string]interface{}{
				stream.AttributeSqsReceiptHandle: "receipt-2",
			},
		},
		{
			Attributes: map[string]interface{}{
				stream.AttributeSqsReceiptHandle: "receipt-3",
			},
		},
	}).Return(nil).Once()

	input.On("Run", context.Background()).Return(nil).Once()
	input.On("Stop").Once()
	input.On("Data").Return(dataChan)

	runPipelineWithSettings(t, &stream.PipelineSettings{
		Interval:  time.Hour,
		BatchSize: 3,
	}, input, 6, context.Background())

	input.AssertExpectations(t)
}
