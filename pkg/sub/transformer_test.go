package sub_test

import (
	"context"
	"errors"
	"github.com/applike/gosoline/pkg/stream"
	"github.com/applike/gosoline/pkg/sub"
	"github.com/applike/gosoline/pkg/sub/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

type testModel struct {
	Bla string
}

func (*testModel) GetId() interface{} {
	return ""
}

func Test_ModelMsgTransformer_MissingTransformerVersion(t *testing.T) {
	tmv := sub.TransformerMapVersion{}
	mmt := sub.BuildTransformer(tmv)

	msg := &stream.ModelMsg{
		Version: 0,
	}

	model, err := mmt(context.TODO(), msg)

	assert.Nil(t, model)
	assert.Error(t, err)
}

func Test_ModelMsgTransformer_ErrorUnmarshallingBody(t *testing.T) {
	mt := new(mocks.ModelTransformer)
	mt.On("GetInput").Return(&testModel{})

	tmv := sub.TransformerMapVersion{
		0: mt,
	}
	mmt := sub.BuildTransformer(tmv)

	msg := &stream.ModelMsg{
		Version: 0,
		Body:    "meh!",
	}

	model, err := mmt(context.TODO(), msg)

	assert.Nil(t, model)
	assert.Error(t, err)

	mt.AssertExpectations(t)
}

func Test_ModelMsgTransformer_TransformError(t *testing.T) {
	recoverableError := errors.New("bla")

	mt := new(mocks.ModelTransformer)
	mt.On("GetInput").Return(&testModel{})
	mt.On("Transform", mock.AnythingOfType("*context.emptyCtx"), mock.Anything).Return(nil, recoverableError)

	tmv := sub.TransformerMapVersion{
		0: mt,
	}
	mmt := sub.BuildTransformer(tmv)

	msg := &stream.ModelMsg{
		Version: 0,
		Body:    "{}",
	}

	model, err := mmt(context.TODO(), msg)

	assert.Nil(t, model)
	assert.Error(t, err)

	mt.AssertExpectations(t)
}

func Test_ModelMsgTransformer_TransformOk(t *testing.T) {
	mt := new(mocks.ModelTransformer)
	mt.On("GetInput").Return(&testModel{})
	mt.On("Transform", mock.AnythingOfType("*context.emptyCtx"), mock.Anything).Return(&testModel{
		Bla: "bla",
	}, nil)

	tmv := sub.TransformerMapVersion{
		0: mt,
	}
	mmt := sub.BuildTransformer(tmv)

	msg := &stream.ModelMsg{
		Version: 0,
		Body:    "{}",
	}

	model, err := mmt(context.TODO(), msg)

	assert.Equal(t, "bla", model.(*testModel).Bla)
	assert.Nil(t, err)

	mt.AssertExpectations(t)
}
