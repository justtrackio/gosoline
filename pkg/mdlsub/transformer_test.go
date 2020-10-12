package mdlsub_test

import (
	"context"
	"errors"
	"github.com/applike/gosoline/pkg/stream"
	"github.com/applike/gosoline/pkg/sub"
	"github.com/applike/gosoline/pkg/sub/mocks"
	"github.com/stretchr/testify/suite"
	"testing"
)

type testInput struct {
	Content string
}

type testModel struct {
	Content string
}

func (*testModel) GetId() interface{} {
	return ""
}

type TransformerTestSuite struct {
	suite.Suite

	ctx                context.Context
	input              *testInput
	model              *testModel
	modelSpecification *sub.ModelSpecification
	modelTransformer   *mocks.ModelTransformer
	transformerMap     sub.TransformerMapVersion
	transformer        sub.ModelMsgTransformer
}

func (s *TransformerTestSuite) SetupTest() {
	s.ctx = context.Background()
	s.input = &testInput{}
	s.model = &testModel{}

	s.modelSpecification = &sub.ModelSpecification{
		CrudType: "create",
		Version:  0,
		ModelId:  "gosoline.test",
	}

	s.modelTransformer = new(mocks.ModelTransformer)
	s.modelTransformer.On("GetInput").Return(s.input)

	s.transformerMap = sub.TransformerMapVersion{
		0: s.modelTransformer,
	}

	s.transformer = sub.BuildTransformer(s.transformerMap)
}

func (s *TransformerTestSuite) Test_ModelMsgTransformer_MissingTransformerVersion() {
	delete(s.transformerMap, 0)
	s.modelTransformer = new(mocks.ModelTransformer)

	msg := &stream.Message{}
	ctx, model, err := s.transformer(s.ctx, s.modelSpecification, msg)

	s.Equal(s.ctx, ctx)
	s.Nil(model)
	s.EqualError(err, "there is no transformer for modelId gosoline.test and version 0")

	s.modelTransformer.AssertExpectations(s.T())
}

func (s *TransformerTestSuite) Test_ModelMsgTransformer_ErrorUnmarshallingBody() {
	msg := &stream.Message{
		Body: "meh!",
	}

	ctx, model, err := s.transformer(s.ctx, s.modelSpecification, msg)

	s.Equal(s.ctx, ctx)
	s.Nil(model)
	s.EqualError(err, "can not decode msg for modelId gosoline.test and version 0: can not decode message body: can not decode message body with encoding 'application/json': invalid character 'm' looking for beginning of value")

	s.modelTransformer.AssertExpectations(s.T())
}

func (s *TransformerTestSuite) Test_ModelMsgTransformer_TransformError() {
	recoverableError := errors.New("reason why transforming failed")
	s.modelTransformer.On("Transform", s.ctx, s.input).Return(nil, recoverableError)

	msg := &stream.Message{
		Body: "{}",
	}

	ctx, model, err := s.transformer(s.ctx, s.modelSpecification, msg)

	s.Equal(s.ctx, ctx)
	s.Nil(model)
	s.EqualError(err, "can not transform body for modelId gosoline.test and version 0: reason why transforming failed")

	s.modelTransformer.AssertExpectations(s.T())
}

func (s *TransformerTestSuite) Test_ModelMsgTransformer_TransformOk() {
	s.input.Content = "text"

	s.modelTransformer.On("Transform", s.ctx, s.input).Return(&testModel{
		Content: "bla",
	}, nil)

	msg := &stream.Message{
		Body: `{"Content": "text"}`,
	}

	ctx, model, err := s.transformer(s.ctx, s.modelSpecification, msg)

	s.Equal(s.ctx, ctx)
	s.Equal("bla", model.(*testModel).Content)
	s.NoError(err)

	s.modelTransformer.AssertExpectations(s.T())
}

func TestTransformerTestSuite(t *testing.T) {
	suite.Run(t, new(TransformerTestSuite))
}
