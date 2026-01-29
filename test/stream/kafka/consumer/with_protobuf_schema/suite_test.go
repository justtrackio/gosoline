//go:build integration && fixtures

package with_protobuf_schema

import (
	"context"
	"testing"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/stream"
	"github.com/justtrackio/gosoline/pkg/test/suite"
	"github.com/justtrackio/gosoline/test/stream/kafka"
	"github.com/justtrackio/gosoline/test/stream/kafka/consumer"
	testEvent "github.com/justtrackio/gosoline/test/stream/kafka/test-event"
	"github.com/twmb/franz-go/pkg/sr"
)

type testSuite struct {
	suite.Suite
	callback       *consumer.CallbackWithSchema
	producer       stream.Producer
	schemaSettings stream.SchemaSettings
}

func TestTestSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

func (s *testSuite) SetupSuite() []suite.Option {
	s.schemaSettings = stream.SchemaSettings{
		Subject: "testEvent",
		Schema:  testEvent.SchemaProto,
		Model:   &testEvent.TestEvent{},
	}
	s.callback = consumer.NewCallbackWithSchema(s.schemaSettings)

	return []suite.Option{
		suite.WithLogLevel(log.LevelDebug),
		suite.WithClockProvider(clock.NewFakeClock(clock.WithNonBlockingSleep)),
		suite.WithConfigFile("../config.dist.yml"),
		suite.WithConfigFile("../config.with_protobuf_schema.yml"),
		kafka.WithKafkaBrokerPort(9193),
		kafka.WithRegisteredSchema(s, s.schemaSettings.Subject, s.schemaSettings.Schema, sr.TypeProtobuf),
		suite.WithConsumer(func(ctx context.Context, config cfg.Config, logger log.Logger) (stream.ConsumerCallback[testEvent.TestEvent], error) {
			return s.callback, nil
		}),
	}
}

func (s *testSuite) SetupTest() (err error) {
	s.producer, err = stream.NewProducer(s.Env().Context(), s.Env().Config(), s.Env().Logger(), "testEvent", stream.WithSchemaSettings(s.schemaSettings))

	return err
}

func (s *testSuite) TestSuccess(app suite.AppUnderTest) {
	s.callback.App = app

	event := &testEvent.TestEvent{
		Id:   1,
		Name: "event 1",
	}

	err := s.producer.WriteOne(s.T().Context(), event)
	s.NoError(err)

	app.WaitDone()

	s.Len(s.callback.ReceivedModels, 1, "the model should have been received once")
}
