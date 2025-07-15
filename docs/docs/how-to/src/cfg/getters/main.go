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
	port, err := config.GetInt("port")
	if err != nil {
		panic(fmt.Errorf("failed to get port: %w", err))
	}
	fmt.Printf("got port: %d\n", port)

	host, err := config.GetString("host")
	if err != nil {
		panic(fmt.Errorf("failed to get host: %w", err))
	}
	fmt.Printf("got host: %s\n", host)
	// highlight-end
}
