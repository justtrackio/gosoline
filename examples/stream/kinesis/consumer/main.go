package main

import (
	"context"
	"encoding/json"

	"github.com/justtrackio/gosoline/pkg/application"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/stream"
)

func main() {
	application.RunConsumer(newConsumer)
}

type consumerCallback struct {
	logger log.Logger
}

func (c consumerCallback) Consume(ctx context.Context, input map[string]string, attributes map[string]string) (bool, error) {
	str, err := json.MarshalIndent(input, "", "    ")
	if err != nil {
		return false, err
	}

	c.logger.WithContext(ctx).Info("Received new message: %s", string(str))

	return true, nil
}

func newConsumer(ctx context.Context, config cfg.Config, logger log.Logger) (stream.ConsumerCallback[map[string]string], error) {
	return consumerCallback{
		logger: logger,
	}, nil
}
