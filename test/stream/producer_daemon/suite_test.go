//go:build integration && fixtures

package producer_daemon

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/test/suite"
)

type ProducerDaemonTestSuite struct {
	suite.Suite
}

func (s *ProducerDaemonTestSuite) SetupSuite() []suite.Option {
	return []suite.Option{
		suite.WithLogLevel("debug"),
		suite.WithConfigFile("config.dist.yml"),
		suite.WithModule("producing-module", NewProducingModule),
	}
}

func (s *ProducerDaemonTestSuite) TestWrite(app suite.AppUnderTest) {
	app.WaitDone()
	output := s.Env().StreamOutput("testEvent")

	firstAggregate := make([]*TestEvent, 0)
	output.UnmarshalAggregate(0, &firstAggregate)
	s.Len(firstAggregate, 3, "the first aggregate should have 3 messages")

	secondAggregate := make([]*TestEvent, 0)
	output.UnmarshalAggregate(1, &secondAggregate)
	s.Len(secondAggregate, 2, "the second aggregate should have 2 messages")

	writtenMessages := output.Len()
	s.Equal(2, writtenMessages)
}

func TestProducerDaemonTestSuite(t *testing.T) {
	suite.Run(t, new(ProducerDaemonTestSuite))
}
