//go:build integration && fixtures
// +build integration,fixtures

package stream_consumer_test

import (
	"testing"

	"github.com/justtrackio/gosoline/examples/more_details/stream-consumer/consumer"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/stream"
	"github.com/justtrackio/gosoline/pkg/test/suite"
)

type ConsumerTestSuite struct {
	suite.Suite
}

func (s *ConsumerTestSuite) SetupSuite() []suite.Option {
	return []suite.Option{
		suite.WithLogLevel("debug"),
		suite.WithConfigFile("../stream-consumer/config.dist.yml"),
		suite.WithModule("consumerModule", stream.NewConsumer("uintConsumer", consumer.NewConsumer())),
	}
}

func (s *ConsumerTestSuite) TestSuccess() *suite.StreamTestCase {
	return &suite.StreamTestCase{
		Input: map[string][]suite.StreamTestCaseInput{
			"consumerInput": {
				{
					Attributes: nil,
					Body:       mdl.Uint(5),
				},
			},
		},
		Assert: func() error {
			var result int
			s.Env().StreamOutput("publisher-outputEvent").Unmarshal(0, &result)

			s.Equal(6, result)

			return nil
		},
	}
}

func TestConsumerTestSuite(t *testing.T) {
	suite.Run(t, new(ConsumerTestSuite))
}
