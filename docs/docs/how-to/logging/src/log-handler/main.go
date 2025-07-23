package main

import (
	"context"
	"fmt"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
)

type MyCustomHandlerSettings struct {
	Channel string `cfg:"channel"`
}

type MyCustomHandler struct {
	channel string
}

func (h *MyCustomHandler) ChannelLevel(name string) (level *int, err error) {
	if name == h.channel {
		return mdl.Box(log.PriorityDebug), nil
	}

	return nil, nil
}

func (h *MyCustomHandler) Level() int {
	return log.PriorityInfo
}

func (h *MyCustomHandler) Log(ctx context.Context, timestamp time.Time, level int, msg string, args []any, err error, data log.Data) error {
	fmt.Printf("%s happened at %s", msg, timestamp.Format(time.RFC822))

	return nil
}

func MyCustomHandlerFactory(config cfg.Config, name string) (log.Handler, error) {
	settings := &MyCustomHandlerSettings{}
	if err := log.UnmarshalHandlerSettingsFromConfig(config, name, settings); err != nil {
		return nil, fmt.Errorf("can not unmarshal handler settings: %w", err)
	}

	return &MyCustomHandler{
		channel: settings.Channel,
	}, nil
}

func main() {
	log.AddHandlerFactory("my-custom-handler", MyCustomHandlerFactory)
}
