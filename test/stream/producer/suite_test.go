//go:build integration && fixtures

package producer

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/application"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/stream"
	"github.com/justtrackio/gosoline/pkg/test/suite"
)

type ProducerTestSuite struct {
	suite.Suite
}

func (s *ProducerTestSuite) SetupSuite() []suite.Option {
	return []suite.Option{
		suite.WithLogLevel(log.LevelDebug),
		suite.WithConfigFile("config.dist.yml"),
		suite.WithAppOptions(
			application.WithLoggerContextFieldsMessageEncoder,
			application.WithTracing,
		),
		suite.WithModule("producing-module", NewProducingModule),
	}
}

func (s *ProducerTestSuite) TestWriteOne(app suite.AppUnderTest) {
	app.WaitDone()

	output := s.Env().StreamOutput("testEvent")
	s.Require().Equal(1, output.Len())

	message, ok := output.Get(0)
	s.Require().True(ok)
	s.Equal("Root=1-5e3d557d-d06c248cc50169bd71b44fec;Parent=af297a5da6453826;Sampled=1", message.Attributes["traceId"])

	loggerContext, ok := message.Attributes[log.MessageAttributeLoggerContext]
	s.Require().True(ok, "the message should contain global logger context fields")
	s.JSONEq(`{"requestId":"test-request-id"}`, loggerContext)

	var event TestEvent
	attributes := output.Unmarshal(0, &event)
	s.Equal(stream.EncodingJson.String(), attributes[stream.AttributeEncoding])
	s.Equal(TestEvent{Id: 1, Name: "test event"}, event)
}

func TestProducerTestSuite(t *testing.T) {
	suite.Run(t, new(ProducerTestSuite))
}
