package main

import (
	"fmt"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
)

type MyCustomHandlerSettings struct {
	Channel Channels `cfg:"channel"`
}

type Channels map[string]string

func MyCustomHandlerFactory(config cfg.Config, name string) (log.Handler, error) {
	settings := &MyCustomHandlerSettings{}
	if err := log.UnmarshalHandlerSettingsFromConfig(config, name, settings); err != nil {
		return nil, fmt.Errorf("can not unmarshal handler settings: %w", err)
	}

	return &MyCustomHandler{
		channels: settings.Channel,
	}, nil
}

type MyCustomHandler struct {
	channels Channels
}

func (h *MyCustomHandler) ChannelLevel(name string) (level *int, err error) {
	levelName, ok := h.channels[name]
	if !ok {
		return nil, nil
	}

	return mdl.Box(log.LevelPriority(levelName)), nil
}

func (h *MyCustomHandler) Level() int {
	return log.PriorityInfo
}

func (h *MyCustomHandler) Log(timestamp time.Time, level int, msg string, args []any, err error, data log.Data) error {
	fmt.Printf("%s happened at %s", msg, timestamp.Format(time.RFC822))

	return nil
}
