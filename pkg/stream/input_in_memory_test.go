package stream_test

import (
	"fmt"
	"testing"

	"github.com/justtrackio/gosoline/pkg/coffin"
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
		err := s.input.Run(s.T().Context())
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

func (s *InMemoryInputTestSuite) TestReset() {
	input := stream.NewInMemoryInput(&stream.InMemorySettings{})
	wait := make(chan struct{})
	cfn := coffin.New(s.T().Context())

	for i := 0; i < 100; i++ {
		cfn.Go(fmt.Sprintf("resetter %d", i), func() error {
			<-wait
			// these two calls should be thread safe and not interfere with each other
			input.Stop()
			input.Reset()

			return nil
		})
	}

	close(wait)

	s.NoError(cfn.Wait())
}

func TestInMemoryInputSuite(t *testing.T) {
	suite.Run(t, new(InMemoryInputTestSuite))
}
