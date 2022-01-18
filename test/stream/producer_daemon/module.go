package producer_daemon

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/stream"
)

type TestEvent struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

type producingModule struct {
	producer stream.Producer
}

func NewProducingModule(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
	var err error
	var producer stream.Producer

	if producer, err = stream.NewProducer(ctx, config, logger, "testEvent"); err != nil {
		return nil, fmt.Errorf("can not create producer testEvent: %w", err)
	}

	return &producingModule{
		producer: producer,
	}, nil
}

func (p producingModule) Run(ctx context.Context) error {
	for i := 0; i < 5; i++ {
		event := &TestEvent{
			Id:   i,
			Name: fmt.Sprintf("event %d", i),
		}

		if err := p.producer.WriteOne(ctx, event); err != nil {
			return nil
		}
	}

	return nil
}
