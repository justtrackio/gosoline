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

func (c consumerCallback) GetModel(_ map[string]string) interface{} {
	return &map[string]string{}
}

func (c consumerCallback) Consume(ctx context.Context, model interface{}, attributes map[string]string) (bool, error) {
	input := model.(*map[string]string)
	str, err := json.MarshalIndent(input, "", "    ")
	if err != nil {
		return false, err
	}

	c.logger.WithContext(ctx).Info("Received new message: %s", string(str))

	return true, nil
}

func newConsumer(ctx context.Context, config cfg.Config, logger log.Logger) (stream.ConsumerCallback, error) {
	return &consumerCallback{
		logger: logger,
	}, nil
}
