package stream_test

import (
	"context"
	"testing"

	"github.com/justtrackio/gosoline/pkg/stream"
	"github.com/justtrackio/gosoline/pkg/stream/mocks"
	"github.com/stretchr/testify/suite"
)

type testContent struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

type ProducerTestSuite struct {
	suite.Suite

	ctx      context.Context
	encoder  stream.MessageEncoder
	output   *mocks.Output
	producer stream.Producer
}

func (s *ProducerTestSuite) SetupTest() {
	s.ctx = context.Background()
	s.encoder = stream.NewMessageEncoder(&stream.MessageEncoderSettings{
		Encoding: stream.EncodingJson,
	})
	s.output = new(mocks.Output)
	s.producer = stream.NewProducerWithInterfaces(s.encoder, s.output)
}

func (s *ProducerTestSuite) TestProducer_WriteOne() {
	content := &testContent{
		Id:   3,
		Name: "foobar",
	}

	expectedMsg := &stream.Message{
		Attributes: map[string]interface{}{
			stream.AttributeEncoding: stream.EncodingJson,
		},
		Body: `{"id":3,"name":"foobar"}`,
	}

	s.output.On("WriteOne", s.ctx, expectedMsg).Return(nil)
	err := s.producer.WriteOne(s.ctx, content)

	s.NoError(err)
	s.output.AssertExpectations(s.T())
}

func (s *ProducerTestSuite) TestProducer_Write() {
	content := []*testContent{
		{
			Id:   3,
			Name: "foobar",
		},
		{
			Id:   5,
			Name: "foobaz",
		},
	}

	expectedMsg := []stream.WritableMessage{
		&stream.Message{
			Attributes: map[string]interface{}{
				stream.AttributeEncoding: stream.EncodingJson,
			},
			Body: `{"id":3,"name":"foobar"}`,
		},
		&stream.Message{
			Attributes: map[string]interface{}{
				stream.AttributeEncoding: stream.EncodingJson,
			},
			Body: `{"id":5,"name":"foobaz"}`,
		},
	}

	s.output.On("Write", s.ctx, expectedMsg).Return(nil)
	err := s.producer.Write(s.ctx, content)

	s.NoError(err)
	s.output.AssertExpectations(s.T())
}

func (s *ProducerTestSuite) TestProducer_Write_SliceError() {
	err := s.producer.Write(s.ctx, "string")

	s.EqualError(err, "can not cast models interface to slice: input is not an slice but instead of type string")
}

func TestProducerTestSuite(t *testing.T) {
	suite.Run(t, new(ProducerTestSuite))
}
