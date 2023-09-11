package main

import (
	"time"

	// 1
	"github.com/justtrackio/gosoline/pkg/cfg"
)

func main() {
	// 2
	config := cfg.New()

	// 3
	options := []cfg.Option{
		cfg.WithConfigFile("config.dist.yml", "yml"),
		cfg.WithConfigMap(map[string]interface{}{
			"enabled": true,
		}),
		cfg.WithConfigSetting("foo", "bar"),
	}

	// 4
	if err := config.Option(options...); err != nil {
		panic(err)
	}
}
