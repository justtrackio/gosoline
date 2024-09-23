//go:build integration

package mdlsub

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/mdlsub"
)

var transformers = mdlsub.TransformerMapTypeVersionFactories{
	"justtrack.gosoline.management.testModel": {
		0: mdlsub.NewGenericTransformer(TestTransformer{}),
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

func (t TestTransformer) GetInput() any {
	return &TestInput{}
}

func (t TestTransformer) Transform(ctx context.Context, inp any) (out mdlsub.Model, err error) {
	mdl := inp.(*TestInput)

	return TestModel{mdl.Id, mdl.Name}, nil
}
