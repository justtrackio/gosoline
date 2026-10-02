//go:build integration

package retry_handler

import (
	"context"
	"testing"

	"github.com/justtrackio/gosoline/pkg/application"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/stream"
	"github.com/justtrackio/gosoline/pkg/test/suite"
)

func TestRetryHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(RetryHandlerTestSuite))
}

type RetryHandlerTestSuite struct {
	suite.Suite
	callback *Callback
}

func (s *RetryHandlerTestSuite) SetupSuite() []suite.Option {
	s.callback = NewCallback()

	return []suite.Option{
		suite.WithLogLevel("debug"),
		suite.WithLogRecording(),
		suite.WithConfigFile("config.dist.yml"),
		suite.WithAppOptions(
			application.WithLoggerContextFieldsMessageEncoder,
			application.WithLoggerContextFieldsResolver(log.ContextFieldsResolver),
			application.WithTracing,
		),
		suite.WithConsumer(func(ctx context.Context, config cfg.Config, logger log.Logger) (stream.ConsumerCallback[DataModel], error) {
			s.callback.logger = logger

			return s.callback, nil
		}),
	}
}

func (s *RetryHandlerTestSuite) TestSuccess(aut suite.AppUnderTest) {
	input := s.Env().StreamInput("consumer")
	s.callback.aut = aut

	input.Publish(DataModel{
		Id:    "3aabc1e4-3c74-47c1-8efb-58f6f862e9a2",
		Title: "my data model",
	}, map[string]string{
		"traceId":                         "Root=1-5e3d557d-d06c248cc50169bd71b44fec;Parent=af297a5da6453826;Sampled=1",
		log.MessageAttributeLoggerContext: `{"requestId":"test-request-id"}`,
	})

	aut.WaitDone()

	s.Require().Len(s.callback.receivedModels, 3, "the model should have been received 3 times")
	records := s.Env().Logs().Channel("consumerCallback")
	s.Require().Len(records, 3, "each delivery should produce a callback log")

	for i, record := range records {
		s.Equal("received retry test message", record.FormattedMsg)
		s.Equal("test-request-id", record.Data.ContextFields["requestId"], "delivery %d should preserve logger context", i+1)
		s.Equal("1-5e3d557d-d06c248cc50169bd71b44fec", record.Data.ContextFields["trace_id"], "delivery %d should preserve the trace", i+1)
	}

	s.Empty(s.callback.receivedAttributes[0], "the first receive should have no attributes")
	s.Equal(s.callback.receivedAttributes[1][stream.AttributeRetry], "true", "the second receive should have the retry attribute")
	s.Equal(s.callback.receivedAttributes[2][stream.AttributeRetry], "true", "the third receive should have the retry attribute")
}
