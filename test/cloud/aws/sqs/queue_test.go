//go:build integration

package sqs_test

import (
	"context"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/cloud/aws/sqs"
	"github.com/justtrackio/gosoline/pkg/test/suite"
)

func TestQueueTestSuite(t *testing.T) {
	suite.Run(t, new(QueueTestSuite))
}

type QueueTestSuite struct {
	suite.Suite
	ctx   context.Context
	queue sqs.Queue
}

func (s *QueueTestSuite) SetupSuite() []suite.Option {
	return []suite.Option{
		suite.WithLogLevel("debug"),
		suite.WithSharedEnvironment(),
		suite.WithConfigFile("client_test_cfg.yml"),
		suite.WithClockProvider(clock.NewRealClock()),
	}
}

func (s *QueueTestSuite) SetupTest() error {
	var err error

	s.ctx = s.Env().Context()
	config := s.Env().Config()
	logger := s.Env().Logger()

	s.queue, err = sqs.NewQueue(s.ctx, config, logger, &sqs.Settings{
		QueueName: "gosoline-test-sqs-queue",
	})

	return err
}

func (s *QueueTestSuite) TestSuccess() {
	ctx, cancel := context.WithTimeout(s.ctx, time.Second)
	defer cancel()

	messages, err := s.queue.Receive(ctx, 10, 1)

	s.Len(messages, 0)
	s.NoError(err)
}
