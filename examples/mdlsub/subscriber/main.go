package main

import (
	"github.com/justtrackio/gosoline/pkg/application"
)

func main() {
	application.RunMdlSubscriber(Transformers,
		application.WithConfigFile("config.dist.yml", "yml"),
	)
}
