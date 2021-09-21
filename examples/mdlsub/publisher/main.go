package main

import (
	"github.com/justtrackio/gosoline/pkg/application"
)

func main() {
	app := application.Default()
	app.Add("publisher", newPublisherModule)
	app.Run()
}
