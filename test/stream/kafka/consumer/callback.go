package consumer

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/stream"
	testEvent "github.com/justtrackio/gosoline/test/stream/kafka/test-event"
)

type Callback struct {
	receivedModels []testEvent.TestEvent
}

func NewCallback() *Callback {
	return &Callback{}
}

func (c *Callback) Consume(_ context.Context, model testEvent.TestEvent, _ map[string]string) (bool, error) {
	c.receivedModels = append(c.receivedModels, model)

	return true, nil
}

func (c *Callback) GetSchemaSettings() (*stream.SchemaSettings, error) {
	return &stream.SchemaSettings{
		Subject: "testEvent",
		Schema:  testEvent.SchemaAvro,
		Model:   testEvent.TestEvent{},
	}, nil
}
