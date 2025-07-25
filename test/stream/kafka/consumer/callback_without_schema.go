package consumer

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/test/suite"
	testEvent "github.com/justtrackio/gosoline/test/stream/kafka/test-event"
)

type CallbackWithoutSchema struct {
	App            suite.AppUnderTest
	ReceivedModels []testEvent.TestEvent
}

func NewCallbackWithoutSchema() *CallbackWithoutSchema {
	return &CallbackWithoutSchema{}
}

func (c *CallbackWithoutSchema) Consume(_ context.Context, model testEvent.TestEvent, _ map[string]string) (bool, error) {
	defer c.App.Stop()
	c.ReceivedModels = append(c.ReceivedModels, model)

	return true, nil
}
