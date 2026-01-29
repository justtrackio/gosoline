//go:build integration && fixtures

package without_schema

import (
	"context"
	"testing"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/ddb"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/mdlsub"
	"github.com/justtrackio/gosoline/pkg/stream"
	"github.com/justtrackio/gosoline/pkg/test/suite"
	"github.com/justtrackio/gosoline/test/stream/kafka"
	"github.com/justtrackio/gosoline/test/stream/kafka/subscriber"
	testEvent "github.com/justtrackio/gosoline/test/stream/kafka/test-event"
)

type testSuite struct {
	suite.Suite
	producer    stream.Producer
	repo        ddb.Repository
	transformer *subscriber.TestEventTransformer
}

func TestTestSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

func (s *testSuite) SetupSuite() []suite.Option {
	s.transformer = subscriber.NewTestEventTransformer()

	return []suite.Option{
		suite.WithLogLevel(log.LevelDebug),
		suite.WithClockProvider(clock.NewFakeClock(clock.WithNonBlockingSleep)),
		suite.WithConfigFile("../config.dist.yml"),
		kafka.WithKafkaBrokerPort(9293),
		suite.WithModuleFactory(func(ctx context.Context, config cfg.Config, logger log.Logger) (map[string]kernel.ModuleFactory, error) {
			return mdlsub.SubscriberFactory(ctx, config, logger, subscriber.TransformerFactories(s.transformer))
		}),
	}
}

func (s *testSuite) SetupTest() (err error) {
	s.producer, err = stream.NewProducer(s.Env().Context(), s.Env().Config(), s.Env().Logger(), "testEvent")
	if err != nil {
		return err
	}

	s.repo, err = s.Env().Localstack("default").DdbRepository(&ddb.Settings{
		ModelId: mdl.ModelId{
			Project:     "justtrack",
			Environment: "test",
			Family:      "gosoline",
			Group:       "kafka",
			Name:        "testModel",
		},
		Main: ddb.MainSettings{
			Model: subscriber.TestModel{},
		},
	})

	return err
}

func (s *testSuite) TestSuccess(app suite.AppUnderTest) {
	s.transformer.App = app

	event := &testEvent.TestEvent{
		Id:   1,
		Name: "event 1",
	}

	err := s.producer.WriteOne(s.T().Context(), event, mdlsub.CreateMessageAttributes(mdl.ModelId{
		Project:     "justtrack",
		Environment: "test",
		Family:      "gosoline",
		Group:       "source-group",
		Name:        "testEvent",
	}, mdlsub.TypeCreate, 0))
	s.NoError(err)

	app.WaitDone()

	actual := &subscriber.TestModel{}
	expected := &subscriber.TestModel{
		Id:   1,
		Name: "event 1",
	}

	qry := s.repo.GetItemBuilder().WithHash(1)
	_, err = s.repo.GetItem(s.Env().Context(), qry, actual)

	s.NoError(err)
	s.Equal(expected, actual)
}
