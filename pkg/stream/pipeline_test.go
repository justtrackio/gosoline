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

type MockAckInput struct {
	input        stream.Input
	acknowledged []*stream.Message
}

func NewMockAckInput(input stream.Input) *MockAckInput {
	return &MockAckInput{input: input, acknowledged: make([]*stream.Message, 0)}
}

func (p *MockAckInput) Run(ctx context.Context) error {
	return p.input.Run(ctx)
}

func (p *MockAckInput) Stop() {
	p.input.Stop()
}

func (p *MockAckInput) Data() chan *stream.Message {
	return p.input.Data()
}

func (p *MockAckInput) Ack(msg *stream.Message) error {
	p.acknowledged = append(p.acknowledged, msg)
	return nil
}

func (p *MockAckInput) AckBatch(msgs []*stream.Message) error {
	p.acknowledged = append(p.acknowledged, msgs...)
	return nil
}

type callback struct {
}

func (c *callback) Boot(config cfg.Config, logger mon.Logger) error {
	return nil
}

func (c *callback) Process(ctx context.Context, messages []*stream.ConsumableMessage) ([]*stream.Message, error) {
	processed := make([]*stream.Message, len(messages))

	for i, msg := range messages {
		processed[i] = &stream.Message{
			Body:       msg.Body + "+B",
			Attributes: msg.Attributes,
			Trace:      msg.Trace,
		}
		msg.Consumed() // mark msg as consumed
	}

	return processed, nil
}

func runPipelineWithSettings(t *testing.T, settings *stream.PipelineSettings, ctx context.Context) {
	logger := mocks.NewLoggerMockedAll()
	metric := mocks.NewMetricWriterMockedAll()

	input := NewMockAckInput(stream.NewFileInputWithInterfaces(logger, stream.FileSettings{
		Filename: "testdata/file_input.json",
	}))
	output := stream.NewOutputMemory()

	callback := &callback{}
	pipe := stream.NewPipeline(callback, callback)

	err := pipe.BootWithInterfaces(logger, metric, input, output, settings)
	assert.NoError(t, err, "the pipeline should boot without an error")

	err = pipe.Run(ctx)
	assert.NoError(t, err, "the pipeline should run without an error")

	size := output.Size()
	assert.Equal(t, 1, size, "the output should contain 1 message")

	assert.Len(t, output.Messages(), 1)
	assert.Equal(t, output.Messages()[0].Body, "A+B+B")

	assert.Len(t, input.acknowledged, 1)
	assert.Equal(t, input.acknowledged[0].Body, "A") // original message was acknowledged
}

func TestPipeline_RunBatchSize(t *testing.T) {
	runPipelineWithSettings(t, &stream.PipelineSettings{
		Interval:  time.Hour,
		BatchSize: 1,
	}, context.Background())
}
