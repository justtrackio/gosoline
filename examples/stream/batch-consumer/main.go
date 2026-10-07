package main

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/application"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/stream"
)

type record struct {
	Value int `json:"value"`
}

func newCallback(_ context.Context, _ cfg.Config, logger log.Logger) (stream.BatchConsumerCallback[record], error) {
	return &callback{logger: logger}, nil
}

type callback struct {
	logger log.Logger
}

func (c *callback) Consume(ctx context.Context, records []record, _ []map[string]string) ([]bool, error) {
	c.logger.Info(ctx, "processing a batch of %d records", len(records))
	acks := make([]bool, len(records))

	for i := range records {
		acks[i] = true
	}

	return acks, nil
}

func main() {
	application.RunBatchConsumer(newCallback, application.WithConfigFile("config.dist.yml", "yml"))
}
