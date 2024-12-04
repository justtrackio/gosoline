package suite_test

import (
	"context"
	"slices"
	"testing"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdlsub"
	"github.com/justtrackio/gosoline/pkg/stream"
	"github.com/justtrackio/gosoline/pkg/test/suite"
	"github.com/justtrackio/gosoline/pkg/uuid"
	"github.com/stretchr/testify/assert"
)

type ConsumerTestSuite struct {
	suite.Suite
	expectedMessageIds []string
	seenMessageIds     []string
	publisher          mdlsub.Publisher
}

type TestMessage struct {
	Id           string `json:"id"`
	ShouldOutput bool   `json:"shouldOutput"`
}

func TestConsumerTestSuite(t *testing.T) {
	var s ConsumerTestSuite
	suite.Run(t, &s)
	assert.Len(t, s.seenMessageIds, len(s.expectedMessageIds))

	slices.Sort(s.expectedMessageIds)
	slices.Sort(s.seenMessageIds)
	assert.Equal(t, s.expectedMessageIds, s.seenMessageIds)
}

func (s *ConsumerTestSuite) SetupSuite() []suite.Option {
	return []suite.Option{
		suite.WithLogLevel("info"),
		suite.WithConfigMap(map[string]any{
			"mdlsub": map[string]any{
				"publishers": map[string]any{
					"echo": map[string]any{
						"output_type": "sqs",
					},
				},
			},
			"stream": map[string]any{
				"input": map[string]any{
					"consumer": map[string]any{
						"type": "sqs",
					},
				},
			},
		}),
		suite.WithConsumer(func(ctx context.Context, config cfg.Config, logger log.Logger) (stream.ConsumerCallback, error) {
			publisher, err := mdlsub.NewPublisher(ctx, config, logger, "echo")
			if err != nil {
				return nil, err
			}

			s.publisher = publisher

			return s, nil
		}),
		suite.WithSharedEnvironment(),
	}
}

func (s *ConsumerTestSuite) GetModel(_ map[string]string) any {
	return &TestMessage{}
}

func (s *ConsumerTestSuite) Consume(ctx context.Context, model any, _ map[string]string) (bool, error) {
	msg := model.(*TestMessage)

	s.seenMessageIds = append(s.seenMessageIds, msg.Id)

	if msg.ShouldOutput {
		if err := s.publisher.Publish(ctx, mdlsub.TypeCreate, 0, msg); err != nil {
			return false, err
		}
	}

	return true, nil
}

func (s *ConsumerTestSuite) TestCaseMap() map[string]*suite.StreamTestCase {
	return map[string]*suite.StreamTestCase{
		"AnotherSingleTest": s.createTestCase(false),
		"EmptyTest":         nil,
	}
}

func (s *ConsumerTestSuite) TestCasesMapWithProvider() map[string]suite.ToStreamTestCase {
	return map[string]suite.ToStreamTestCase{
		"AnotherSingleTest":   s.createTestCase(true),
		"AnotherWithProvider": suite.ToStreamTestCase(s.createTestCase(true)),
		"Nil":                 nil,
	}
}

func (s *ConsumerTestSuite) TestCasesEmptyMap() map[string]*suite.StreamTestCase {
	return map[string]*suite.StreamTestCase{}
}

func (s *ConsumerTestSuite) TestSingleTest() *suite.StreamTestCase {
	return s.createTestCase(false)
}

func (s *ConsumerTestSuite) TestSkipped() *suite.StreamTestCase {
	return nil
}

func (s *ConsumerTestSuite) TestNilProvider() suite.ToStreamTestCase {
	return nil
}

func (s *ConsumerTestSuite) TestProvider() suite.ToStreamTestCase {
	return s.createTestCase(true)
}

func (s *ConsumerTestSuite) createTestCase(hasOutput bool) *suite.StreamTestCase {
	messageId := uuid.New().NewV4()
	s.expectedMessageIds = append(s.expectedMessageIds, messageId)

	var output map[string][]suite.StreamTestCaseOutput
	if hasOutput {
		output = map[string][]suite.StreamTestCaseOutput{
			"publisher-echo": {
				{
					Model: &TestMessage{},
					ExpectedAttributes: map[string]string{
						"encoding": "application/json",
						"modelId":  "justtrack.gosoline.test.echo",
						"type":     "create",
						"version":  "0",
					},
					ExpectedBody: &TestMessage{
						Id:           messageId,
						ShouldOutput: true,
					},
				},
			},
		}
	}

	return &suite.StreamTestCase{
		Input: map[string][]suite.StreamTestCaseInput{
			"consumer": {
				{
					Body: TestMessage{
						Id:           messageId,
						ShouldOutput: hasOutput,
					},
				},
			},
		},
		Output: output,
		Assert: func() error {
			s.Contains(s.seenMessageIds, messageId)

			return nil
		},
	}
}
