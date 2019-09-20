package kernel

import (
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"os"
)

func New(application string) *kernel {
	config := cfg.New(application)

	tags := mon.Tags{
		"application": config.GetString("app_name"),
	}

	logger := mon.NewLogger(config, tags)

	return NewWithInterfaces(config, logger)
}

func NewWithInterfaces(config cfg.Config, logger mon.Logger) *kernel {
	return &kernel{
		sig: make(chan os.Signal, 1),

		config: config,
		logger: logger.WithChannel("kernel"),

		factories: make([]ModuleFactory, 0),
	}
}
