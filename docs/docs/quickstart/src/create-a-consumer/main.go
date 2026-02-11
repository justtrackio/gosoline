package main

import "github.com/justtrackio/gosoline/pkg/application"

func main() {
	application.RunConsumer(NewConsumer,
		application.WithConfigFile("config.dist.yml", "yml"),
	)
}
