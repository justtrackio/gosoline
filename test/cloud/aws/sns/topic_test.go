//go:build integration

package sns_test

import (
	"context"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/cloud/aws/sns"
	"github.com/justtrackio/gosoline/pkg/test/suite"
)

func TestTopicTestSuite(t *testing.T) {
	suite.Run(t, new(TopicTestSuite))
}

type TopicTestSuite struct {
	suite.Suite
	ctx    context.Context
	topic1 sns.Topic
	topic2 sns.Topic
}

func (s *TopicTestSuite) SetupSuite() []suite.Option {
	return []suite.Option{
		suite.WithLogLevel("debug"),
		suite.WithSharedEnvironment(),
		suite.WithConfigFile("client_test_cfg.yml"),
		suite.WithClockProvider(clock.NewRealClock()),
	}
}

func (s *TopicTestSuite) SetupTest() error {
	var err error

	s.ctx = s.Env().Context()
	config := s.Env().Config()
	logger := s.Env().Logger()

	s.topic1, err = sns.NewTopic(s.ctx, config, logger, &sns.TopicSettings{
		TopicName: "test",
	})
	if err != nil {
		return err
	}

	s.topic2, err = sns.NewTopic(s.ctx, config, logger, &sns.TopicSettings{
		TopicName: "test",
	})
	if err != nil {
		return err
	}

	return err
}

func (s *TopicTestSuite) TestSuccess() {
	ctx, cancel := context.WithTimeout(s.ctx, time.Second)
	defer cancel()

	err := s.topic1.Publish(ctx, "message")
	s.NoError(err)

	err = s.topic2.Publish(ctx, "message")
	s.NoError(err)
}
