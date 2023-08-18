package main

import (
	// 1
	"github.com/justtrackio/gosoline/pkg/cfg"
)

func main() {
	// 2
	config := cfg.New()

	// 3
	options := []cfg.Option{
		cfg.WithConfigSetting("host", "localhost"),
		cfg.WithConfigSetting("port", "80"),
	}

	// 4
	if err := config.Option(options...); err != nil {
		panic(err)
	}
}
