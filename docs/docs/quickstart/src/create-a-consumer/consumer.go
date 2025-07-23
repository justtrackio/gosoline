package main

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/stream"
)

type Input struct {
	Id   string `json:"id"`
	Body string `json:"body"`
}

type Consumer struct {
	logger log.Logger
}

func NewConsumer(ctx context.Context, config cfg.Config, logger log.Logger) (stream.ConsumerCallback[Input], error) {
	return &Consumer{
		logger: logger,
	}, nil
}

func (c Consumer) Consume(ctx context.Context, input Input, attributes map[string]string) (bool, error) {
	c.logger.Info(ctx, "got input with id %q and body %q", input.Id, input.Body)

	return true, nil
}
