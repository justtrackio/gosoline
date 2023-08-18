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
		cfg.WithEnvKeyReplacer(cfg.DefaultEnvKeyReplacer),
	}

	// 4
	if err := config.Option(options...); err != nil {
		panic(err)
	}
}
