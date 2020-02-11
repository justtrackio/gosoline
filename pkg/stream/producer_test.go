package stream_test

import (
	"context"
	"github.com/applike/gosoline/pkg/stream"
	"github.com/applike/gosoline/pkg/stream/mocks"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestProducer_Write(t *testing.T) {
	ctx := context.Background()
	content := "this is a test"

	encoder := stream.NewMessageEncoder(&stream.MessageEncoderSettings{
		Encoding: stream.EncodingJson,
	})

	expectedMsg := &stream.Message{
		Attributes: map[string]interface{}{
			stream.AttributeEncoding: stream.EncodingJson,
		},
		Body: `"this is a test"`,
	}

	output := new(mocks.Output)
	output.On("WriteOne", ctx, expectedMsg).Return(nil)

	producer := stream.NewProducerWithInterfaces(encoder, output)
	err := producer.WriteOne(ctx, content)

	assert.NoError(t, err)
}
