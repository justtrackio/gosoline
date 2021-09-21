package stream_test

import (
	"context"
	"testing"

	"github.com/justtrackio/gosoline/pkg/stream"
	"github.com/stretchr/testify/suite"
)

type InMemoryInputTestSuite struct {
	suite.Suite
	input *stream.InMemoryInput
}

func (s *InMemoryInputTestSuite) SetupTest() {
	s.input = stream.ProvideInMemoryInput("test", &stream.InMemorySettings{
		Size: 3,
	})
}

func (s *InMemoryInputTestSuite) TestRun() {
	msg := stream.NewMessage("content")

	go func() {
		err := s.input.Run(context.Background())
		s.NoError(err)
	}()

	s.input.Publish(msg)
	s.input.Stop()

	readMessages := make([]*stream.Message, 0)

	for msg := range s.input.Data() {
		readMessages = append(readMessages, msg)
	}

	s.Len(readMessages, 1, "1 message should have been read")
	s.Equal("content", msg.Body, "message body should contain content")
}

func TestInMemoryInputSuite(t *testing.T) {
	suite.Run(t, new(InMemoryInputTestSuite))
}
