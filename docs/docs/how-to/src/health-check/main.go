package main

import "github.com/justtrackio/gosoline/pkg/application"

func main() {
	application.RunModule("hello-world", NewHelloWorldModule,
		application.WithConfigFile("config.dist.yml", "yml"),
		application.WithHttpHealthCheck,
	)
}
