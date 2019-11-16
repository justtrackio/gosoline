package application

import "github.com/applike/gosoline/pkg/stream"

func RunConsumer(callback stream.ConsumerCallback, options ...Option) {
	consumers := stream.NewConsumerFactory(map[string]stream.ConsumerCallback{
		"default": callback,
	})

	app := Default(options...)
	app.AddFactory(consumers)
	app.Run()
}
