package main

import "github.com/justtrackio/gosoline/pkg/application"

func main() {
	application.Run(
		application.WithConfigFile("config.dist.yml", "yml"),
		application.WithModuleFactory("hello-world", NewHelloWorldModule),
	)
}
