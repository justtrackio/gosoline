//go:build integration && fixtures

package consumer

import (
	"context"
	"testing"

	"github.com/justtrackio/gosoline/pkg/application"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/stream"
	"github.com/justtrackio/gosoline/pkg/test/suite"
	"github.com/justtrackio/gosoline/pkg/tracing"
)

type ConsumerTestSuite struct {
	suite.Suite
	callback *callback
}

func (s *ConsumerTestSuite) SetupSuite() []suite.Option {
	return []suite.Option{
		suite.WithLogLevel(log.LevelDebug),
		suite.WithLogRecording(),
		suite.WithConfigFile("config.dist.yml"),
		suite.WithAppOptions(
			application.WithLoggerContextFieldsMessageEncoder,
			application.WithLoggerContextFieldsResolver(log.ContextFieldsResolver),
			application.WithTracing,
		),
		suite.WithConsumer(func(_ context.Context, _ cfg.Config, logger log.Logger) (stream.ConsumerCallback[TestEvent], error) {
			s.callback = &callback{logger: logger}

			return s.callback, nil
		}),
	}
}

func (s *ConsumerTestSuite) TestConsume(app suite.AppUnderTest) {
	s.callback.app = app
	event := TestEvent{Id: 1, Name: "test event"}
	s.Env().StreamInput("testEvent").Publish(event, map[string]string{
		stream.AttributeEncoding:          stream.EncodingJson.String(),
		log.MessageAttributeLoggerContext: `{"requestId":"test-request-id"}`,
		"traceId":                         "Root=1-5e3d557d-d06c248cc50169bd71b44fec;Parent=af297a5da6453826;Sampled=1",
	})
	app.WaitDone()

	s.Equal(event, s.callback.receivedModel)
	s.Equal(map[string]any{"requestId": "test-request-id"}, s.callback.loggerFields)
	s.Equal(&tracing.Trace{
		TraceId:  "1-5e3d557d-d06c248cc50169bd71b44fec",
		ParentId: "af297a5da6453826",
		Sampled:  true,
	}, s.callback.trace)
	s.NotContains(s.callback.receivedAttributes, log.MessageAttributeLoggerContext)
	s.NotContains(s.callback.receivedAttributes, "traceId")
	s.Equal(stream.EncodingJson.String(), s.callback.receivedAttributes[stream.AttributeEncoding])

	records := s.Env().Logs().Channel("consumerCallback")
	s.Require().Len(records, 1)
	s.Equal("consumed test event", records[0].FormattedMsg)
	s.Equal("test-request-id", records[0].Data.ContextFields["requestId"])
	s.Equal("1-5e3d557d-d06c248cc50169bd71b44fec", records[0].Data.ContextFields["trace_id"])
}

func TestConsumerTestSuite(t *testing.T) {
	suite.Run(t, new(ConsumerTestSuite))
}
