package stream_test

import (
	"context"
	"sync"
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

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		err := s.input.Run(context.Background())
		s.NoError(err)
		wg.Done()
	}()

	s.input.Publish(msg)
	s.input.Stop()
	wg.Wait()

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
	cfn := coffin.New(func(cfn coffin.StartingCoffin) {
		for i := 0; i < 100; i++ {
			cfn.Go(func() error {
				<-wait
				// these two calls should be thread safe and not interfere with each other
				input.Stop()
				input.Reset()

				return nil
			})
		}
	})

	close(wait)

	s.NoError(cfn.Wait())
}

func TestInMemoryInputSuite(t *testing.T) {
	suite.Run(t, new(InMemoryInputTestSuite))
}
