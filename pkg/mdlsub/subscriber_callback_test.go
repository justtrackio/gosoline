package mdlsub_test

import (
	"context"
	"testing"

	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/mdlsub"
	"github.com/justtrackio/gosoline/pkg/mdlsub/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

func TestSubscriberCallbackTestSuite(t *testing.T) {
	suite.Run(t, new(SubscriberCallbackTestSuite))
}

type SubscriberCallbackTestSuite struct {
	suite.Suite
	core       *mocks.SubscriberCore
	output     *mocks.Output
	callback   *mdlsub.SubscriberCallback
	modelId    mdl.ModelId
	attributes map[string]string
}

func (s *SubscriberCallbackTestSuite) SetupTest() {
	s.core = mocks.NewSubscriberCore(s.T())
	s.output = mocks.NewOutput(s.T())

	s.modelId = mdl.ModelId{
		Project: "justtrack",
		Family:  "gosoline",
		Group:   "mdlsub",
		Name:    "testModel",
	}

	sourceModel := mdlsub.SubscriberModel{
		ModelId: s.modelId,
	}

	s.core.EXPECT().GetModelIds().Return([]string{"justtrack.gosoline.mdlsub.testModel"}).Once()

	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(s.T()))
	s.callback = mdlsub.NewSubscriberCallbackWithInterfaces(
		logger,
		s.core,
		sourceModel,
	)

	// Default attributes for a valid message
	s.attributes = map[string]string{
		"modelId": "justtrack.gosoline.mdlsub.testModel",
		"type":    "create",
		"version": "1",
	}
}

func (s *SubscriberCallbackTestSuite) TestConsume_Success() {
	input := &TestInput{Id: 42}
	expectedModel := TestModel{Id: 42}

	transformer := mdlsub.EraseTransformerTypes[TestInput, TestModel](TestTransformer{})

	spec := &mdlsub.ModelSpecification{
		ModelId:  "justtrack.gosoline.mdlsub.testModel",
		CrudType: "create",
		Version:  1,
	}

	s.core.EXPECT().GetTransformer(spec).Return(transformer, nil).Once()
	s.core.EXPECT().GetOutput(spec).Return(s.output, nil).Once()

	s.output.EXPECT().Persist(mock.Anything, expectedModel, "create").Return(nil).Once()

	ack, err := s.callback.Consume(s.T().Context(), input, s.attributes)

	s.NoError(err)
	s.True(ack)
}

func (s *SubscriberCallbackTestSuite) TestConsume_UnknownModelId() {
	input := &TestInput{Id: 42}

	s.attributes["modelId"] = "invalid.model.id"

	spec := &mdlsub.ModelSpecification{
		ModelId:  "invalid.model.id",
		CrudType: "create",
		Version:  1,
	}

	s.core.EXPECT().GetTransformer(spec).Return(nil, mdlsub.NewUnknownModelError("invalid.model.id")).Once()

	ack, err := s.callback.Consume(s.T().Context(), input, s.attributes)

	s.Error(err)
	s.False(ack)
	s.True(mdlsub.IsUnknownModelError(err))
}

func (s *SubscriberCallbackTestSuite) TestConsume_UnknownVersion() {
	input := &TestInput{Id: 42}

	s.attributes["version"] = "99"

	spec := &mdlsub.ModelSpecification{
		ModelId:  "justtrack.gosoline.mdlsub.testModel",
		CrudType: "create",
		Version:  99,
	}

	s.core.EXPECT().GetTransformer(spec).Return(nil, mdlsub.NewUnknownModelVersionError("justtrack.gosoline.mdlsub.testModel", 99)).Once()

	ack, err := s.callback.Consume(s.T().Context(), input, s.attributes)

	// Now returns an error for unknown version
	s.Error(err)
	s.False(ack)
	s.True(mdlsub.IsUnknownModelVersionError(err))
}

func (s *SubscriberCallbackTestSuite) TestConsume_TransformReturnsNil() {
	input := &TestInput{Id: 42}

	// Create a transformer that returns nil
	nilTransformer := mdlsub.EraseTransformerTypes[TestInput, TestModel](NilTestTransformer{})

	spec := &mdlsub.ModelSpecification{
		ModelId:  "justtrack.gosoline.mdlsub.testModel",
		CrudType: "create",
		Version:  1,
	}

	s.core.EXPECT().GetTransformer(spec).Return(nilTransformer, nil).Once()

	ack, err := s.callback.Consume(s.T().Context(), input, s.attributes)

	// Should acknowledge when transformer returns nil (skipping the item)
	s.NoError(err)
	s.True(ack)
}

// NilTestTransformer is a transformer that returns nil to test skipping behavior
type NilTestTransformer struct{}

func (t NilTestTransformer) Transform(ctx context.Context, inp TestInput) (out *TestModel, err error) {
	return nil, nil
}
