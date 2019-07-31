package stream_test

import (
	"github.com/applike/gosoline/pkg/stream"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMessageBuilder_GetMessage(t *testing.T) {
	msg, err := BuildSqsTestMessage()

	assert.NoError(t, err)
	assert.Containsf(t, msg.Attributes, stream.AttributeSqsDelaySeconds, "the message has no sqs delay attribute")
	assert.JSONEq(t, `{"Foo":"bar"}`, msg.Body)
}

func BuildSqsTestMessage() (*stream.Message, error) {
	type BodyStruct struct {
		Foo string
	}

	body := BodyStruct{
		Foo: "bar",
	}

	builder := stream.NewMessageBuilder()
	builder.WithBody(body)
	builder.WithSqsDelaySeconds(45)

	return builder.GetMessage()
}
