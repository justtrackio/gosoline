package subscriber

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/mdlsub"
	"github.com/justtrackio/gosoline/pkg/stream"
	"github.com/justtrackio/gosoline/pkg/test/suite"
	testEvent "github.com/justtrackio/gosoline/test/stream/kafka/test-event"
)

var TransformerFactories = func(transformer mdlsub.TypedTransformer[testEvent.TestEvent, TestModel]) mdlsub.TransformerMapTypeVersionFactories {
	return mdlsub.TransformerMapTypeVersionFactories{
		"justtrack.gosoline.source-group.testEvent": {
			0: mdlsub.NewGenericTransformer(transformer),
		},
	}
}

type TestModel struct {
	Id   int    `json:"id" ddb:"key=hash"`
	Name string `json:"name"`
}

func (m TestModel) GetId() any {
	return m.Id
}

func NewTestEventTransformer() *TestEventTransformer {
	return &TestEventTransformer{}
}

type TestEventTransformer struct {
	App suite.AppUnderTest
}

func (t TestEventTransformer) Transform(_ context.Context, inp testEvent.TestEvent) (out *TestModel, err error) {
	defer t.App.Stop()

	return &TestModel{
		Id:   inp.Id,
		Name: inp.Name,
	}, nil
}

func NewTestEventTransformerWithSchema(schemaSettings stream.SchemaSettings) *TestEventTransformerWithSchema {
	return &TestEventTransformerWithSchema{
		schemaSettings: schemaSettings,
	}
}

type TestEventTransformerWithSchema struct {
	schemaSettings stream.SchemaSettings
	App            suite.AppUnderTest
}

func (t TestEventTransformerWithSchema) Transform(_ context.Context, inp testEvent.TestEvent) (out *TestModel, err error) {
	defer t.App.Stop()

	return &TestModel{
		Id:   inp.Id,
		Name: inp.Name,
	}, nil
}

func (t TestEventTransformerWithSchema) GetSchemaSettings() (*stream.SchemaSettings, error) {
	return &t.schemaSettings, nil
}
