//go:build integration && fixtures

package mdlsub

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/ddb"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/mdlsub"
	"github.com/justtrackio/gosoline/pkg/test/suite"
)

func TestFixturesTestSuite(t *testing.T) {
	suite.Run(t, new(FixturesTestSuite))
}

type FixturesTestSuite struct {
	suite.Suite
	repo ddb.Repository
}

func (s *FixturesTestSuite) SetupSuite() []suite.Option {
	return []suite.Option{
		suite.WithLogLevel("debug"),
		suite.WithConfigFile("config.dist.yml"),
		suite.WithModuleFactory(mdlsub.NewSubscriberFactory(transformers)),
		suite.WithFixtureSetFactories(mdlsub.FixtureSetFactory(transformers)),
		suite.WithEnvSetup(func() error {
			wiremockAddress := s.Env().Wiremock("wiremock").Address()
			config := s.Env().Config()

			return config.Option(cfg.WithConfigMap(map[string]any{
				"fixtures.providers.default.host": wiremockAddress,
			}))
		}),
	}
}

func (s *FixturesTestSuite) SetupTest() (err error) {
	s.repo, err = s.Env().DynamoDb("default").Repository(&ddb.Settings{
		ModelId: mdl.ModelId{
			Project:     "justtrack",
			Environment: "test",
			Family:      "gosoline",
			Group:       "mdlsub",
			Name:        "testModel",
		},
		Main: ddb.MainSettings{
			Model: TestModel{},
		},
	})

	return
}

func (s *FixturesTestSuite) TestSuccess(app suite.AppUnderTest) {
	app.Stop()

	act := &TestModel{}
	exp := &TestModel{
		Id:   1,
		Name: "foo",
	}

	ctx := s.Env().Context()
	qry := s.repo.GetItemBuilder().WithHash(1)
	_, err := s.repo.GetItem(ctx, qry, act)

	s.NoError(err)
	s.Equal(exp, act)
}
