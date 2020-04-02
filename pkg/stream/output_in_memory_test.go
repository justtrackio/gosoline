package stream_test

import (
	"context"
	"github.com/applike/gosoline/pkg/stream"
	"github.com/stretchr/testify/suite"
	"testing"
)

type InMemoryOutputTestSuite struct {
	suite.Suite
	output *stream.InMemoryOutput
}

func (s *InMemoryOutputTestSuite) SetupTest() {
	s.output = stream.ProvideInMemoryOutput("test")
}

func (s *InMemoryOutputTestSuite) TestWrite() {
	msg := stream.NewMessage("content")
	err := s.output.WriteOne(context.Background(), msg)

	s.NoError(err, "there should be no error on write")
	s.Equal(1, s.output.Len(), "there should be 1 written message")

	written, ok := s.output.Get(0)

	s.True(ok, "it should be possible to get 1 message")
	s.Equal("content", written.Body, "the body of the message should match")
}

func TestInMemoryOutputTestSuite(t *testing.T) {
	suite.Run(t, new(InMemoryOutputTestSuite))
}
