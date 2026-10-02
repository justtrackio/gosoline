package producer

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/stream"
	"github.com/justtrackio/gosoline/pkg/tracing"
)

// TestEvent is the payload written by the producer application.
type TestEvent struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

type producingModule struct {
	producer stream.Producer
}

// NewProducingModule creates an application module that writes one test event.
func NewProducingModule(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
	producer, err := stream.NewProducer(ctx, config, logger, "testEvent")
	if err != nil {
		return nil, fmt.Errorf("can not create producer testEvent: %w", err)
	}

	return &producingModule{producer: producer}, nil
}

func (p producingModule) Run(ctx context.Context) error {
	ctx = tracing.ContextWithTrace(ctx, &tracing.Trace{
		TraceId:  "1-5e3d557d-d06c248cc50169bd71b44fec",
		Id:       "af297a5da6453826",
		ParentId: "0123456789abcdef",
		Sampled:  true,
	})
	ctx = log.AppendGlobalContextFields(ctx, map[string]any{
		"requestId": "test-request-id",
	})

	if err := p.producer.WriteOne(ctx, &TestEvent{Id: 1, Name: "test event"}); err != nil {
		return fmt.Errorf("can not write test event: %w", err)
	}

	return nil
}
