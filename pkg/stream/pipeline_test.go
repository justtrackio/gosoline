package stream_test

import (
	"context"
	"testing"
	"time"

	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	metricMocks "github.com/justtrackio/gosoline/pkg/metric/mocks"
	"github.com/justtrackio/gosoline/pkg/stream"
	"github.com/stretchr/testify/assert"
)

type callback struct{}

func (c *callback) Process(ctx context.Context, messages []*stream.Message) ([]*stream.Message, error) {
	for _, msg := range messages {
		msg.Body = "foobaz"
	}

	return messages, nil
}

func runPipelineWithSettings(t *testing.T, settings *stream.PipelineSettings, ctx context.Context) {
	logger := logMocks.NewLoggerMockedAll()
	metric := metricMocks.NewWriterMockedAll()
	output := stream.NewInMemoryOutput()

	input := stream.NewInMemoryInput(&stream.InMemorySettings{Size: 1})
	input.Publish(&stream.Message{
		Body: "foobar",
	})
	input.Stop()

	callback := &callback{}

	pipe, err := stream.NewPipelineWithInterfaces(logger, metric, input, output, settings, callback)
	assert.NoError(t, err)

	err = pipe.Run(ctx)
	assert.NoError(t, err, "the pipeline should run without an error")

	size := output.Size()
	assert.Equal(t, 1, size, "the output should contain 1 message")

	contains := output.ContainsBody("foobaz")
	assert.True(t, contains, "the output should contain the body 'foobaz'")
}

func TestPipeline_RunBatchSize(t *testing.T) {
	runPipelineWithSettings(t, &stream.PipelineSettings{
		Interval:  time.Hour,
		BatchSize: 1,
	}, context.Background())
}
