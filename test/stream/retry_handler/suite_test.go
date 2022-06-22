//go:build integration

package retry_handler

import (
	"context"
	"testing"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/stream"
	"github.com/justtrackio/gosoline/pkg/test/suite"
)

type RetryHandlerTestSuite struct {
	suite.Suite
	callback *Callback
}

func (s *RetryHandlerTestSuite) SetupSuite() []suite.Option {
	s.callback = NewCallback()

	return []suite.Option{
		suite.WithLogLevel("debug"),
		suite.WithConfigFile("config.dist.yml"),
		suite.WithConsumer(func(ctx context.Context, config cfg.Config, logger log.Logger) (stream.ConsumerCallback, error) {
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
	}, map[string]interface{}{})

	aut.WaitDone()

	s.Len(s.callback.receivedModels, 3, "the model should have been received 3 times")
	s.Empty(s.callback.receivedAttributes[0], "the first receive should have no attributes")
	s.Equal(s.callback.receivedAttributes[1][stream.AttributeRetry], true, "the second receive should have the retry attribute")
	s.Equal(s.callback.receivedAttributes[2][stream.AttributeRetry], true, "the third receive should have the retry attribute")
}

func TestRetryHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(RetryHandlerTestSuite))
}
