package main

import (
	"github.com/justtrackio/gosoline/examples/more_details/stream-consumer/consumer"
	"github.com/justtrackio/gosoline/pkg/application"
)

func main() {
	application.RunConsumer(consumer.NewConsumer())
}
