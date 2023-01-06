package mdlsub_test

import (
	"context"
	"testing"

	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/mdlsub"
	streamMocks "github.com/justtrackio/gosoline/pkg/stream/mocks"
	"github.com/stretchr/testify/suite"
)

type PublisherTestSuite struct {
	suite.Suite
	producer  *streamMocks.Producer
	publisher mdlsub.Publisher
}

func (s *PublisherTestSuite) SetupTest() {
	logger := logMocks.NewLoggerMockedAll()
	s.producer = new(streamMocks.Producer)

	s.publisher = mdlsub.NewPublisherWithInterfaces(logger, s.producer, &mdlsub.PublisherSettings{
		ModelId: mdl.ModelId{
			Project:     "gosoline",
			Family:      "test",
			Group:       "grp",
			Application: "app",
			Name:        "event",
		},
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
		"type":    mdlsub.TypeCreate,
		"version": 0,
		"modelId": "gosoline.test.grp.event",
	}

	s.producer.On("WriteOne", ctx, event, expectedAttributes).Return(nil)

	err := s.publisher.Publish(ctx, mdlsub.TypeCreate, 0, event)
	s.NoError(err)
}

func TestPublisherTestSuite(t *testing.T) {
	suite.Run(t, new(PublisherTestSuite))
}
