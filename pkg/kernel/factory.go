package kernel

import (
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"os"
)

func New(config cfg.Config, logger mon.Logger) *kernel {
	return &kernel{
		sig: make(chan os.Signal, 1),

		config: config,
		logger: logger.WithChannel("kernel"),

		factories: make([]ModuleFactory, 0),
	}
}

func NewWithDefault(application string, tagsFromConfig mon.TagsFromConfig) *kernel {
	config := cfg.NewWithDefaultClients(application)

	tags := mon.Tags{
		"application": config.GetString("app_name"),
	}

	logger := mon.NewLogger(config, tags, tagsFromConfig)

	return New(config, logger)
}
