package log

import (
	"fmt"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
)

type Handler interface {
	Channels() []string
	Level() int
	Log(timestamp time.Time, level int, msg string, args []interface{}, err error, data Data) error
}

type HandlerFactory func(config cfg.Config, name string) (Handler, error)

var handlerFactories = map[string]HandlerFactory{}

func AddHandlerFactory(typ string, factory HandlerFactory) {
	handlerFactories[typ] = factory
}

func NewHandlersFromConfig(config cfg.Config) ([]Handler, error) {
	settings := &LoggerSettings{}
	config.UnmarshalKey("log", settings)

	var i int
	var ok bool
	var err error
	var handlerFactory HandlerFactory
	handlers := make([]Handler, len(settings.Handlers))

	for name, handlerSettings := range settings.Handlers {
		if handlerFactory, ok = handlerFactories[handlerSettings.Type]; !ok {
			return nil, fmt.Errorf("there is no logging handler of type %s", handlerSettings.Type)
		}

		if handlers[i], err = handlerFactory(config, name); err != nil {
			return nil, fmt.Errorf("can not create logging handler of type %s on index %d: %w", handlerSettings.Type, i, err)
		}

		i++
	}

	return handlers, nil
}

func UnmarshalHandlerSettingsFromConfig(config cfg.Config, name string, settings interface{}) {
	handlerConfigKey := getHandlerConfigKey(name)
	config.UnmarshalKey(handlerConfigKey, settings, cfg.UnmarshalWithDefaultsFromKey("log.level", "level"))
}

func getHandlerConfigKey(name string) string {
	return fmt.Sprintf("log.handlers.%s", name)
}
