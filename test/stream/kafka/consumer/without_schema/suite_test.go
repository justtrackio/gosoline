//go:build integration && fixtures

package without_schema

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
)

type testSuite struct {
	suite.Suite
	callback *consumer.CallbackWithoutSchema
	producer stream.Producer
}

func TestTestSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

func (s *testSuite) SetupSuite() []suite.Option {
	s.callback = consumer.NewCallbackWithoutSchema()

	return []suite.Option{
		suite.WithLogLevel(log.LevelDebug),
		suite.WithClockProvider(clock.NewFakeClock(clock.WithNonBlockingSleep)),
		suite.WithConfigFile("../config.dist.yml"),
		kafka.WithKafkaBrokerPort(9194),
		suite.WithConsumer(func(ctx context.Context, config cfg.Config, logger log.Logger) (stream.ConsumerCallback[testEvent.TestEvent], error) {
			return s.callback, nil
		}),
	}
}

func (s *testSuite) SetupTest() (err error) {
	s.producer, err = stream.NewProducer(s.Env().Context(), s.Env().Config(), s.Env().Logger(), "testEvent")

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
