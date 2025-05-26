package main

import (
	"fmt"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

type MyCustomHandlerSettings struct {
	Channel log.Channels `cfg:"channel"`
}

func MyCustomHandlerFactory(config cfg.Config, name string) (log.Handler, error) {
	settings := &MyCustomHandlerSettings{}
	log.UnmarshalHandlerSettingsFromConfig(config, name, settings)

	return &MyCustomHandler{
		channels: settings.Channel,
	}, nil
}

type MyCustomHandler struct {
	channels log.Channels
}

func (h *MyCustomHandler) Channels() log.Channels {
	return h.channels
}

func (h *MyCustomHandler) Level() int {
	return log.PriorityInfo
}

func (h *MyCustomHandler) Log(timestamp time.Time, level int, msg string, args []interface{}, err error, data log.Data) error {
	fmt.Printf("%s happenend at %s", msg, timestamp.Format(time.RFC822))
	return nil
}
