package consumer

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/test/suite"
	"github.com/justtrackio/gosoline/pkg/tracing"
)

// TestEvent is the payload received by the consumer application.
type TestEvent struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

type callback struct {
	app                suite.AppUnderTest
	logger             log.Logger
	receivedModel      TestEvent
	receivedAttributes map[string]string
	loggerFields       map[string]any
	trace              *tracing.Trace
}

func (c *callback) Consume(ctx context.Context, model TestEvent, attributes map[string]string) (bool, error) {
	defer c.app.Stop()

	c.receivedModel = model
	c.receivedAttributes = attributes
	c.loggerFields = log.GlobalContextFieldsResolver(ctx)
	c.trace = tracing.GetTraceFromContext(ctx)
	c.logger.Info(ctx, "consumed test event")

	return true, nil
}
