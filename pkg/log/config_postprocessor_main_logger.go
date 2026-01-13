package log

import (
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
)

func init() {
	cfg.AddPostProcessor(8, "gosoline.log.handler_main", MainLoggerConfigPostProcessor)
}

// MainLoggerConfigPostProcessor ensures a "main" logger handler is configured.
// If not explicitly defined, it defaults to an "iowriter" handler.
func MainLoggerConfigPostProcessor(config cfg.GosoConf) (bool, error) {
	if config.IsSet("log.handlers.main.type") {
		return false, nil
	}

	configOptions := []cfg.Option{
		cfg.WithConfigSetting("log.handlers.main.type", "iowriter"),
	}

	if err := config.Option(configOptions...); err != nil {
		return false, fmt.Errorf("can not apply config settings for main handler: %w", err)
	}

	return true, nil
}
