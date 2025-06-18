package mdlsub_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/jarcoal/httpmock"
	"github.com/justtrackio/gosoline/pkg/fixtures"
	loggerMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/mdlsub"
	mdlsubMocks "github.com/justtrackio/gosoline/pkg/mdlsub/mocks"
	"github.com/stretchr/testify/suite"
)

func TestFixtureSetTestSuite(t *testing.T) {
	suite.Run(t, new(FixtureSetTestSuite))
}

type FixtureSetTestSuite struct {
	suite.Suite

	source     mdlsub.SubscriberModel
	spec       *mdlsub.ModelSpecification
	core       *mdlsubMocks.SubscriberCore
	output     *mdlsubMocks.Output
	fixtureSet fixtures.FixtureSet
}

func (s *FixtureSetTestSuite) SetupTest() {
	s.core = new(mdlsubMocks.SubscriberCore)
	s.output = new(mdlsubMocks.Output)

	s.source = mdlsub.SubscriberModel{
		ModelId: mdl.ModelId{
			Project: "justtrack",
			Family:  "gosoline",
			Group:   "mdlsub",
			Name:    "testModel",
		},
	}

	s.spec = &mdlsub.ModelSpecification{
		CrudType: "create",
		Version:  1,
		ModelId:  "justtrack.gosoline.mdlsub.testModel",
	}

	settings := &mdlsub.FixtureSettings{
		Dataset: mdlsub.FixtureSettingsDataset{
			Id: 1,
		},
		Host: "http://localhost:8080",
		Path: "path/for/mdlsub",
	}

	client := resty.New()
	httpmock.ActivateNonDefault(client.GetClient())

	logger := loggerMocks.NewLoggerMock(loggerMocks.WithMockAll)
	s.fixtureSet = mdlsub.NewFixtureSetWithInterfaces(logger, s.source, s.core, settings, client)
}

func (s *FixtureSetTestSuite) TearDownTest() {
	httpmock.DeactivateAndReset()

	s.core.AssertExpectations(s.T())
	s.output.AssertExpectations(s.T())
}

func (s *FixtureSetTestSuite) TestSuccess() {
	ctx := context.Background()

	s.output.EXPECT().Persist(ctx, TestModel{Id: 1}, "create").Return(nil)
	s.output.EXPECT().Persist(ctx, TestModel{Id: 2}, "create").Return(nil)

	s.core.EXPECT().GetLatestModelIdVersion(s.source.ModelId).Return(1, nil)
	s.core.EXPECT().GetTransformer(s.spec).Return(mdlsub.EraseTransformerTypes(TestTransformer{}), nil)
	s.core.EXPECT().GetOutput(s.spec).Return(s.output, nil)

	httpmock.RegisterResponder("GET", "http://localhost:8080/path/for/mdlsub?dataset_id=1&model_id=justtrack.gosoline.mdlsub.testModel&version=1",
		func(request *http.Request) (*http.Response, error) {
			resp := httpmock.NewStringResponse(200, `{"data":[{"id":1},{"id":2}]}`)
			resp.Header.Add("Content-Type", "application/json")

			return resp, nil
		},
	)

	err := s.fixtureSet.Write(ctx)
	s.NoError(err)
}
