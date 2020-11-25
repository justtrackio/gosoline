package application

import (
	"github.com/applike/gosoline/pkg/mdlsub"
	"github.com/applike/gosoline/pkg/stream"
)

func RunConsumer(callback stream.ConsumerCallbackFactory, options ...Option) {
	consumers := stream.NewConsumerFactory(map[string]stream.ConsumerCallbackFactory{
		"default": callback,
	})

	app := Default(options...)
	app.AddFactory(consumers)
	app.Run()
}

func RunMdlSubscriber(transformers mdlsub.TransformerMapTypeVersionFactories) {
	app := Default()

	subs := mdlsub.NewSubscriberFactory(transformers)
	app.AddFactory(subs)

	app.Run()
}
