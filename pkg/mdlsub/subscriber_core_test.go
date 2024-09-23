package mdlsub_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/mdlsub"
	"github.com/justtrackio/gosoline/pkg/mdlsub/mocks"
	"github.com/stretchr/testify/suite"
)

func TestSubscriberCoreTestSuite(t *testing.T) {
	suite.Run(t, new(SubscriberCoreTestSuite))
}

type SubscriberCoreTestSuite struct {
	suite.Suite
	core    mdlsub.SubscriberCore
	modelId mdl.ModelId
}

func (s *SubscriberCoreTestSuite) SetupTest() {
	s.core = mdlsub.NewSubscriberCoreWithInterfaces(transformers, outputs)
	s.modelId = mdl.ModelId{
		Project: "justtrack",
		Family:  "gosoline",
		Group:   "mdlsub",
		Name:    "testModel",
	}
}

func (s *SubscriberCoreTestSuite) TestGetLatestModelIdVersion() {
	version, err := s.core.GetLatestModelIdVersion(s.modelId)
	s.NoError(err)
	s.Equal(2, version)
}

func (s *SubscriberCoreTestSuite) TestGetLatestModelIdMissingModelId() {
	_, err := s.core.GetLatestModelIdVersion(mdl.ModelId{
		Project: "justtrack",
		Family:  "gosoline",
		Group:   "foobar",
		Name:    "testModel",
	})
	s.EqualError(err, "failed to find model transformer for model id justtrack.gosoline.foobar.testModel")
}

func (s *SubscriberCoreTestSuite) TestGetLatestModelIdMissingVersions() {
	transformers := mdlsub.ModelTransformers{
		"justtrack.gosoline.mdlsub.testModel": mdlsub.VersionedModelTransformers{},
	}
	core := mdlsub.NewSubscriberCoreWithInterfaces(transformers, outputs)
	_, err := core.GetLatestModelIdVersion(s.modelId)
	s.EqualError(err, "there are no versions available for transformer model id justtrack.gosoline.mdlsub.testModel")
}

var transformers = mdlsub.ModelTransformers{
	"justtrack.gosoline.mdlsub.testModel": mdlsub.VersionedModelTransformers{
		0: &TestTransformer{},
		1: &TestTransformer{},
		2: &TestTransformer{},
	},
}

var outputs = mdlsub.Outputs{
	"justtrack.gosoline.mdlsub.testModel": map[int]mdlsub.Output{
		0: new(mocks.Output),
		1: new(mocks.Output),
		2: new(mocks.Output),
	},
}
