package application

import (
	"github.com/applike/gosoline/pkg/mdlsub"
	"github.com/applike/gosoline/pkg/stream"
	"github.com/applike/gosoline/pkg/sub"
)

func RunConsumer(callback stream.ConsumerCallback, options ...Option) {
	consumers := stream.NewConsumerFactory(map[string]stream.ConsumerCallback{
		"default": callback,
	})

	app := Default(options...)
	app.AddFactory(consumers)
	app.Run()
}

func RunSubscriber(transformers sub.TransformerMapTypeVersionFactories, options ...Option) {
	subs := sub.NewSubscriberFactory(transformers)

	app := Default(options...)
	app.AddFactory(subs)
	app.Run()
}

func RunMdlSubscriber(transformers mdlsub.TransformerMapTypeVersionFactories) {
	app := Default()

	subs := mdlsub.NewSubscriberFactory(transformers)
	app.AddFactory(subs)

	app.Run()
}
