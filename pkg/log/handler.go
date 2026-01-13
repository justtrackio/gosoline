package log

import (
	"context"
	"fmt"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
)

// Handler defines the interface for log output destinations (e.g., console, file, Sentry).
// Implementations are responsible for writing the formatted log entry to the underlying resource.
type Handler interface {
	ChannelLevel(name string) (level *int, err error)
	Level() int
	Log(ctx context.Context, timestamp time.Time, level int, msg string, args []any, err error, data Data) error
}

// HandlerFactory is a function type for creating new handlers from configuration.
type HandlerFactory func(config cfg.Config, name string) (Handler, error)

var handlerFactories = map[string]HandlerFactory{}

// AddHandlerFactory registers a new factory function for creating log handlers of a specific type.
// This allows for extending the logging system with custom handler implementations.
func AddHandlerFactory(typ string, factory HandlerFactory) {
	handlerFactories[typ] = factory
}

// NewHandlersFromConfig creates a slice of log handlers based on the provided configuration.
// It parses the "log.handlers" section of the config and instantiates the corresponding handlers.
func NewHandlersFromConfig(config cfg.Config) ([]Handler, error) {
	settings := &LoggerSettings{}
	if err := config.UnmarshalKey("log", settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal logger settings: %w", err)
	}

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

// UnmarshalHandlerSettingsFromConfig extracts settings for a specific named handler from the configuration.
// It applies defaults where necessary, particularly for the log level.
func UnmarshalHandlerSettingsFromConfig(config cfg.Config, name string, settings any) error {
	handlerConfigKey := getHandlerConfigKey(name)
	if err := config.UnmarshalKey(handlerConfigKey, settings, cfg.UnmarshalWithDefaultsFromKey("log.level", "level")); err != nil {
		return fmt.Errorf("failed to unmarshal handler settings for key %q: %w", handlerConfigKey, err)
	}

	return nil
}

func getHandlerConfigKey(name string) string {
	return fmt.Sprintf("log.handlers.%s", name)
}
