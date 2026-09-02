package stream_test

import (
	"context"
	"testing"
	"time"

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
	s.input.Reset()
}

func (s *InMemoryInputTestSuite) TestRun() {
	ctx := s.T().Context()
	msg := stream.NewMessage("content")
	processed := make(chan *stream.Message, 1)
	cfn := coffin.New()

	cfn.Go(func() error {
		return s.input.Run(ctx, func(_ context.Context, msg *stream.Message) bool {
			processed <- msg

			return true
		})
	})

	s.input.Publish(msg)
	processedMsg := <-processed
	s.input.Stop(ctx)

	s.NoError(cfn.Wait())
	s.Equal("content", processedMsg.Body, "message body should contain content")
}

func (s *InMemoryInputTestSuite) TestRunConcurrently() {
	ctx := s.T().Context()
	input := stream.NewInMemoryInput(&stream.InMemorySettings{
		Size:        2,
		RunnerCount: 2,
	})
	processing := make(chan struct{}, 2)
	release := make(chan struct{})
	cfn := coffin.New()

	cfn.Go(func() error {
		return input.Run(ctx, func(context.Context, *stream.Message) bool {
			processing <- struct{}{}
			<-release

			return true
		})
	})

	input.Publish(stream.NewMessage("first"), stream.NewMessage("second"))

	for range 2 {
		select {
		case <-processing:
		case <-time.After(time.Second):
			close(release)
			input.Stop(ctx)
			s.NoError(cfn.Wait())
			s.FailNow("messages were not processed concurrently")
		}
	}

	close(release)
	input.Stop(ctx)

	s.NoError(cfn.Wait())
}

func (s *InMemoryInputTestSuite) TestStopDrainsBufferedMessages() {
	ctx := s.T().Context()

	// stopping the input must not drop messages which were published before. as both the stopped channel and
	// the message channel are ready at the same time, we have to repeat this a few times to reliably catch a
	// regression instead of relying on how select picks a ready case.
	for i := range 50 {
		input := stream.NewInMemoryInput(&stream.InMemorySettings{
			Size: 3,
		})
		processed := make(chan *stream.Message, 3)
		cfn := coffin.New()

		cfn.Go(func() error {
			return input.Run(ctx, func(_ context.Context, msg *stream.Message) bool {
				processed <- msg

				return true
			})
		})

		input.Publish(stream.NewMessage("first"), stream.NewMessage("second"))
		input.Stop(ctx)

		s.NoError(cfn.Wait())
		s.Len(processed, 2, "all buffered messages should have been processed in run %d", i)
	}
}

func (s *InMemoryInputTestSuite) TestReset() {
	ctx := s.T().Context()
	input := stream.NewInMemoryInput(&stream.InMemorySettings{})
	wait := make(chan struct{})
	cfn := coffin.New()

	for i := 0; i < 100; i++ {
		cfn.Go(func() error {
			<-wait
			// these two calls should be thread safe and not interfere with each other
			input.Stop(ctx)
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
