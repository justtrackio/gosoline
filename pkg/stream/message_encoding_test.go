package stream_test

import (
	"context"
	"github.com/applike/gosoline/pkg/stream"
	"github.com/jonboulle/clockwork"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

type encodingTestStruct struct {
	Id        int       `json:"id"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"createdAt"`
}

func TestMessageEncoder_Encode(t *testing.T) {
	clock := clockwork.NewFakeClock()

	data := encodingTestStruct{
		Id:        3,
		Text:      "example",
		CreatedAt: clock.Now(),
	}

	encoder := stream.NewMessageEncoder(&stream.MessageEncoderConfig{
		Encoding: stream.EncodingJson,
	})

	msg, err := encoder.Encode(context.Background(), data, map[string]interface{}{
		"attribute1": 5,
		"attribute2": "test",
	})

	assert.NoError(t, err)
	assert.JSONEq(t, `{"id":3,"text":"example","createdAt":"1984-04-04T00:00:00Z"}`, msg.Body)

	assert.Contains(t, msg.Attributes, "attribute1")
	assert.Equal(t, 5, msg.Attributes["attribute1"])

	assert.Contains(t, msg.Attributes, "attribute2")
	assert.Equal(t, "test", msg.Attributes["attribute2"])
}
