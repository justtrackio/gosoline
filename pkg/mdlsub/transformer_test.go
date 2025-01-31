package mdlsub_test

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/mdlsub"
)

type TestInput struct {
	Id int `json:"id"`
}

type TestModel struct {
	Id int `json:"id"`
}

type TestTransformer struct{}

func (t TestTransformer) GetInput() any {
	return &TestInput{}
}

func (t TestTransformer) GetModel() any {
	return &TestModel{}
}

func (t TestTransformer) Transform(ctx context.Context, inp any) (out mdlsub.Model, err error) {
	mdl := inp.(*TestInput)

	return TestModel{mdl.Id}, nil
}

func (m TestModel) GetId() any {
	return m.Id
}
