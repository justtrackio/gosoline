package main

import (
	"github.com/justtrackio/gosoline/pkg/application"
)

func main() {
	application.Run(
		application.WithModuleFactory("publisher", newPublisherModule),
	)
}
