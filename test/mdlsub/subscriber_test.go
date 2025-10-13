//go:build integration && fixtures

package mdlsub

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/mdlsub"
	"github.com/justtrackio/gosoline/pkg/test/suite"
)

func TestSubscriberTestSuite(t *testing.T) {
	suite.Run(t, new(SubscriberTestSuite))
}

type SubscriberTestSuite struct {
	suite.Suite
}

func (s *SubscriberTestSuite) SetupSuite() []suite.Option {
	return []suite.Option{
		suite.WithLogLevel("debug"),
		suite.WithConfigFile("config.dist.yml"),
		suite.WithModuleFactory(mdlsub.NewSubscriberFactory(transformers)),
	}
}

func (s *SubscriberTestSuite) TestSuccess() (suite.SubscriberTestCase, error) {
	return suite.DdbTestCase(suite.DdbSubscriberTestCase{
		Name:          "testModel",
		SourceModelId: "justtrack.gosoline.management.testModel",
		TargetModelId: "justtrack.gosoline.mdlsub.testModel",
		Version:       0,
		Input: TestInput{
			Id:   1337,
			Name: "foo",
		},
		Assert: func(t *testing.T, fetcher *suite.DdbSubscriberFetcher) {
			mdl := &TestModel{}
			fetcher.ByHash(1337, mdl)

			s.Equal(1337, mdl.Id)
			s.Equal("foo", mdl.Name)
		},
	})
}
