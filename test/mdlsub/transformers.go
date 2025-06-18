//go:build integration

package mdlsub

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/mdlsub"
)

var transformers = mdlsub.TransformerMapTypeVersionFactories{
	"justtrack.gosoline.management.testModel": {
		0: mdlsub.NewGenericTransformer[TestInput, TestModel](TestTransformer{}),
	},
}

type TestInput struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

type TestModel struct {
	Id   int    `json:"id" ddb:"key=hash"`
	Name string `json:"name"`
}

func (m TestModel) GetId() any {
	return m.Id
}

type TestTransformer struct{}

func (t TestTransformer) Transform(ctx context.Context, inp TestInput) (out *TestModel, err error) {
	return &TestModel{
		Id:   inp.Id,
		Name: inp.Name,
	}, nil
}
