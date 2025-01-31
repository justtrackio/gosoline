//go:build fixtures

package suite_test

import (
	"context"
	"testing"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdlsub"
	"github.com/justtrackio/gosoline/pkg/test/suite"
)

type TestModel struct {
	Id   int
	Text string
}

type SubscriberTestSuite struct {
	suite.Suite
}

func TestSubscriberTestSuite(t *testing.T) {
	suite.Run(t, &SubscriberTestSuite{})
}

func (m TestModel) GetId() any {
	return m.Id
}

func (s *SubscriberTestSuite) SetupSuite() []suite.Option {
	return []suite.Option{
		suite.WithLogLevel("info"),
		suite.WithConfigMap(map[string]any{
			"kvstore": map[string]any{
				"testModel": map[string]any{
					"type": "chain",
					"elements": []string{
						"inMemory",
						"ddb",
					},
				},
			},
			"mdlsub": map[string]any{
				"subscribers": map[string]any{
					"testModel": map[string]any{
						"output": "kvstore",
					},
				},
			},
			"test": map[string]any{
				"components": map[string]any{
					"ddb": map[string]any{
						"default": map[string]any{},
					},
				},
			},
		}),
		suite.WithSubscribers(map[string]mdlsub.TransformerMapVersionFactories{
			"justtrack.gosoline.test.testModel": {
				0: func(ctx context.Context, config cfg.Config, logger log.Logger) (mdlsub.ModelTransformer, error) {
					return s, nil
				},
			},
		}),
		suite.WithSharedEnvironment(),
	}
}

func (s *SubscriberTestSuite) GetInput() any {
	return &TestInput{}
}

func (s *SubscriberTestSuite) GetModel() any {
	return &TestModel{}
}

func (s *SubscriberTestSuite) Transform(_ context.Context, inp any) (out mdlsub.Model, err error) {
	input := inp.(*TestInput)

	return &TestModel{
		Id:   1,
		Text: input.Text,
	}, nil
}

func (s *SubscriberTestSuite) TestInput() (suite.SubscriberTestCase, error) {
	return suite.KvstoreTestCase(suite.KvstoreSubscriberTestCase{
		Name:    "testModel",
		ModelId: "justtrack.gosoline.test.testModel",
		Input: &TestInput{
			Text: "this is a test",
		},
		Assert: func(t *testing.T, fetcher *suite.KvstoreSubscriberFetcher) {
			actual := &TestModel{}
			fetcher.Get(1, actual)

			expected := &TestModel{
				Id:   1,
				Text: "this is a test",
			}

			s.Equal(expected, actual)
		},
	})
}
