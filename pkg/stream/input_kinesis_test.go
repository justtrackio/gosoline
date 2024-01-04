package stream_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/stream"
	"github.com/stretchr/testify/assert"
)

func TestKinesisMessageHandler(t *testing.T) {
	c := make(chan *stream.Message, 10)
	h := stream.NewKinesisMessageHandler(c)

	err := h.Handle([]byte(`{"attributes":{"type":"message"},"body":"foo"}`))
	assert.NoError(t, err)
	err = h.Handle([]byte("not a message"))
	assert.Error(t, err)

	h.Done()

	msgs := make([]*stream.Message, 0)
	for msg := range c {
		msgs = append(msgs, msg)
	}

	assert.Equal(t, []*stream.Message{
		{
			Attributes: map[string]string{
				"type": "message",
			},
			Body: "foo",
		},
	}, msgs)
}
