package mdlsub_test

import (
	"context"
)

type TestInput struct {
	Id int `json:"id"`
}

type TestModel struct {
	Id int `json:"id"`
}

type TestTransformer struct{}

func (t TestTransformer) Transform(ctx context.Context, inp TestInput) (out *TestModel, err error) {
	return &TestModel{
		Id: inp.Id,
	}, nil
}

func (m TestModel) GetId() any {
	return m.Id
}
