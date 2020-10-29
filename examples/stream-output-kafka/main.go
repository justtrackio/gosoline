package main

import (
	"github.com/applike/gosoline/pkg/application"
)

func main() {
	app := application.Default()
	app.Add("producer", new(KafkaProducer))
	app.Run()
}
