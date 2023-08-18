package main

import (
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
)

func main() {
	config := cfg.New()

	options := []cfg.Option{
		cfg.WithConfigFile("config.dist.yml", "yml"),
	}

	if err := config.Option(options...); err != nil {
		panic(err)
	}

	// highlight-start
	fmt.Printf("got port: %d\n", config.GetInt("port"))
	fmt.Printf("got host: %s\n", config.GetString("host"))
	// highlight-end
}
