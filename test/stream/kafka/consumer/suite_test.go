//go:build integration && fixtures

package consumer

import (
	"context"
	"testing"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/stream"
	"github.com/justtrackio/gosoline/pkg/test/suite"
	"github.com/justtrackio/gosoline/test/stream/kafka"
	"github.com/justtrackio/gosoline/test/stream/kafka/producer"
	testEvent "github.com/justtrackio/gosoline/test/stream/kafka/test-event"
	"github.com/twmb/franz-go/pkg/sr"
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

	schemaSettings := stream.SchemaSettings{
		Subject: "testEvent",
		Schema:  testEvent.SchemaAvro,
		Model:   &testEvent.TestEvent{},
	}

	return []suite.Option{
		suite.WithLogLevel(log.LevelDebug),
		suite.WithConfigFile("config.dist.yml"),
		kafka.WithRegisteredSchema(s, "testEvent", testEvent.SchemaAvro, sr.TypeAvro),
		suite.WithModule("producer-module", producer.NewProducerModule(1, stream.WithSchemaSettings(schemaSettings))),
		suite.WithConsumer(func(ctx context.Context, config cfg.Config, logger log.Logger) (stream.ConsumerCallback[testEvent.TestEvent], error) {
			return s.callback, nil
		}),
	}
}

func (s *RetryHandlerTestSuite) TestSuccess(aut suite.AppUnderTest) {
	aut.WaitDone()

	s.Len(s.callback.receivedModels, 1, "the model should have been received once")
}
