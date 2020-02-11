package stream_test

import (
	"context"
	"github.com/applike/gosoline/pkg/stream"
	"github.com/jonboulle/clockwork"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"testing"
	"time"
)

type encodingTestStruct struct {
	Id        int       `json:"id"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"createdAt"`
}

type MessageEncoderSuite struct {
	suite.Suite
	clock   clockwork.Clock
	encoder stream.MessageEncoder
}

func (s *MessageEncoderSuite) SetupTest() {
	s.clock = clockwork.NewFakeClock()
	s.encoder = stream.NewMessageEncoder(&stream.MessageEncoderSettings{
		Encoding: stream.EncodingJson,
	})
}

func (s *MessageEncoderSuite) TestMessageEncoder_Encode() {
	data := encodingTestStruct{
		Id:        3,
		Text:      "example",
		CreatedAt: s.clock.Now(),
	}

	msg, err := s.encoder.Encode(context.Background(), data, map[string]interface{}{
		"attribute1": 5,
		"attribute2": "test",
	})

	assert.NoError(s.T(), err)
	assert.JSONEq(s.T(), `{"id":3,"text":"example","createdAt":"1984-04-04T00:00:00Z"}`, msg.Body)

	assert.Contains(s.T(), msg.Attributes, "attribute1")
	assert.Equal(s.T(), 5, msg.Attributes["attribute1"])

	assert.Contains(s.T(), msg.Attributes, "attribute2")
	assert.Equal(s.T(), "test", msg.Attributes["attribute2"])
}

func (s *MessageEncoderSuite) TestMessageEncoder_Decode() {
	msg := &stream.Message{
		Attributes: map[string]interface{}{
			stream.AttributeEncoding: stream.EncodingJson,
			"attribute1":             5,
			"attribute2":             "test",
		},
		Body: `{"id":3,"text":"example","createdAt":"1984-04-04T00:00:00Z"}`,
	}

	data := &encodingTestStruct{}
	_, attributes, err := s.encoder.Decode(context.Background(), msg, data)

	expected := &encodingTestStruct{
		Id:        3,
		Text:      "example",
		CreatedAt: s.clock.Now(),
	}

	assert.NoError(s.T(), err)
	assert.Equal(s.T(), expected, data)
	assert.NotContains(s.T(), attributes, stream.AttributeEncoding)
	assert.Contains(s.T(), attributes, "attribute1")
	assert.Equal(s.T(), 5, attributes["attribute1"])
	assert.Contains(s.T(), attributes, "attribute2")
	assert.Equal(s.T(), "test", attributes["attribute2"])
}

func TestMessageEncoderSuite(t *testing.T) {
	suite.Run(t, new(MessageEncoderSuite))
}
