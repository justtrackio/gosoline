package pubsub_test

import (
	"context"
	monMocks "github.com/applike/gosoline/pkg/mon/mocks"
	"github.com/applike/gosoline/pkg/pubsub"
	streamMocks "github.com/applike/gosoline/pkg/stream/mocks"
	"github.com/stretchr/testify/suite"
	"testing"
)

type PublisherTestSuite struct {
	suite.Suite
	producer  *streamMocks.Producer
	publisher pubsub.Publisher
}

func (s *PublisherTestSuite) SetupTest() {
	logger := monMocks.NewLoggerMockedAll()
	s.producer = new(streamMocks.Producer)

	s.publisher = pubsub.NewPublisherWithInterfaces(logger, s.producer, &pubsub.PublisherSettings{
		Project:     "gosoline",
		Family:      "test",
		Application: "app",
		Name:        "event",
	})
}

func (s *PublisherTestSuite) TestPublish() {
	type testEvent struct {
		Id    int    `json:"id"`
		Title string `json:"title"`
	}

	ctx := context.Background()
	event := testEvent{
		Id:    1,
		Title: "my title",
	}

	expectedAttributes := map[string]interface{}{
		"type":    pubsub.TypeCreate,
		"version": 0,
		"modelId": "gosoline.test.app.event",
	}

	s.producer.On("WriteOne", ctx, event, expectedAttributes).Return(nil)

	err := s.publisher.Publish(ctx, pubsub.TypeCreate, 0, event)
	s.NoError(err)
}

func TestPublisherTestSuite(t *testing.T) {
	suite.Run(t, new(PublisherTestSuite))
}
