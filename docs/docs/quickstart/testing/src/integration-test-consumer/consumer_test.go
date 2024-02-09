package consumertest

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/stream"
	"github.com/justtrackio/gosoline/pkg/test/suite"
)

type ConsumerTestSuite struct {
	suite.Suite
}

func TestConsumerTestSuite(t *testing.T) {
	suite.Run(t, new(ConsumerTestSuite))
}

func (s *ConsumerTestSuite) SetupSuite() []suite.Option {
	return []suite.Option{
		suite.WithConfigFile("config.dist.yml"),
		suite.WithConsumer(NewConsumer),
	}
}

func (s *ConsumerTestSuite) TestSuccess() *suite.StreamTestCase {
	return &suite.StreamTestCase{
		Input: map[string][]suite.StreamTestCaseInput{
			"consumer": {
				{
					Body: &Todo{
						Id:   1,
						Text: "do it",
					},
				},
			},
		},
		Output: map[string][]suite.StreamTestCaseOutput{
			"todos": {
				{
					Model: &Todo{},
					ExpectedBody: &Todo{
						Id:     1,
						Text:   "do it",
						Status: "pending",
					},
					ExpectedAttributes: map[string]string{
						stream.AttributeEncoding: stream.EncodingJson.String(),
					},
				},
			},
		},
		Assert: func() error {
			msgCount := s.Env().StreamOutput("todos").Len()
			s.Equal(1, msgCount)

			return nil
		},
	}
}
