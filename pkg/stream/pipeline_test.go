package stream_test

import (
	"context"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/mon/mocks"
	"github.com/applike/gosoline/pkg/stream"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

type callback struct {
}

func (c *callback) Boot(config cfg.Config, logger mon.Logger) error {
	return nil
}

func (c *callback) Process(ctx context.Context, messages []*stream.Message) ([]*stream.Message, error) {
	for _, msg := range messages {
		msg.Body = "foobaz"
	}

	return messages, nil
}

func runPipelineWithSettings(t *testing.T, settings *stream.PipelineSettings, ctx context.Context) {
	logger := mocks.NewLoggerMockedAll()
	metric := mocks.NewMetricWriterMockedAll()

	input := stream.NewFileInputWithInterfaces(logger, stream.FileSettings{
		Filename: "testdata/file_input.json",
	})
	output := stream.NewOutputMemory()

	callback := &callback{}
	pipe := stream.NewPipeline(callback)

	err := pipe.BootWithInterfaces(logger, metric, input, output, settings)
	assert.NoError(t, err, "the pipeline should boot without an error")

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
