package consumer

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/stream"
	"github.com/justtrackio/gosoline/pkg/test/suite"
	testEvent "github.com/justtrackio/gosoline/test/stream/kafka/test-event"
)

type CallbackWithSchema struct {
	App            suite.AppUnderTest
	ReceivedModels []testEvent.TestEvent
	schemaSettings stream.SchemaSettings
}

func NewCallbackWithSchema(schemaSettings stream.SchemaSettings) *CallbackWithSchema {
	return &CallbackWithSchema{
		schemaSettings: schemaSettings,
	}
}

func (c *CallbackWithSchema) Consume(_ context.Context, model testEvent.TestEvent, _ map[string]string) (bool, error) {
	defer c.App.Stop()
	c.ReceivedModels = append(c.ReceivedModels, model)

	return true, nil
}

func (c *CallbackWithSchema) GetSchemaSettings() (*stream.SchemaSettings, error) {
	return &c.schemaSettings, nil
}
