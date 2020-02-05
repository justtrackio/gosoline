package stream_test

import (
	"github.com/applike/gosoline/pkg/stream"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewMessage(t *testing.T) {
	msg := stream.NewMessage(`{"foo": "bar"}`, map[string]interface{}{
		"attribute1": 2,
		"attribute2": "value",
	})

	expectedMsg := &stream.Message{
		Attributes: map[string]interface{}{
			"attribute1": 2,
			"attribute2": "value",
		},
		Body: `{"foo": "bar"}`,
	}

	assert.Equal(t, expectedMsg, msg)
}

func TestNewJsonMessage(t *testing.T) {
	msg := stream.NewJsonMessage(`{"foo": "bar"}`, map[string]interface{}{
		"attribute1": 2,
		"attribute2": "value",
	})

	expectedMsg := &stream.Message{
		Attributes: map[string]interface{}{
			"attribute1":             2,
			"attribute2":             "value",
			stream.AttributeEncoding: stream.EncodingJson,
		},
		Body: `{"foo": "bar"}`,
	}

	assert.Equal(t, expectedMsg, msg)
}
