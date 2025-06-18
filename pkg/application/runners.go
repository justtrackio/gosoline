package application

import (
	"context"
	"fmt"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/justtrackio/gosoline/pkg/httpserver"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdlsub"
	"github.com/justtrackio/gosoline/pkg/stream"
)

func Run(options ...Option) {
	app := Default(options...)
	app.Run()
}

func RunFunc(f func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.ModuleRunFunc, error), options ...Option) {
	options = append(options, WithModuleFactory("func", func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
		var err error
		var run kernel.ModuleRunFunc

		if run, err = f(ctx, config, logger); err != nil {
			return nil, fmt.Errorf("can not build run function: %w", err)
		}

		return kernel.NewModuleFunc(run), nil
	}))

	Run(options...)
}

func RunModule(name string, moduleFactory kernel.ModuleFactory, options ...Option) {
	options = append(options, WithModuleFactory(name, moduleFactory))

	Run(options...)
}

func RunHttpDefaultServer(definer httpserver.Definer, options ...Option) {
	RunHttpServers(
		map[string]httpserver.Definer{
			"default": definer,
		},
		options...,
	)
}

func RunHttpServers(servers map[string]httpserver.Definer, options ...Option) {
	options = append(options, WithExecBackoffSettings(&exec.BackoffSettings{
		InitialInterval: time.Millisecond * 100,
		MaxElapsedTime:  time.Second * 10,
		MaxInterval:     time.Second,
	}))

	for name, definer := range servers {
		options = append(options, WithModuleFactory("httpserver-"+name, httpserver.New(name, definer)))
	}

	Run(options...)
}

// RunConsumer runs the provided consumer as an application. You can pass additional options to customize the way it is executed.
func RunConsumer[M any](callback stream.ConsumerCallbackFactory[M], options ...Option) {
	RunConsumers[M](stream.ConsumerCallbackMap[M]{
		"default": callback,
	}, options...)
}

// RunUntypedConsumer runs the provided untyped consumer as an application. You can pass additional options to customize the way it is executed.
//
// Prefer using RunConsumer if possible as it provided additional type safety (especially, if you are only expecting a single type as input anyway).
func RunUntypedConsumer(callback stream.UntypedConsumerCallbackFactory, options ...Option) {
	RunUntypedConsumers(stream.UntypedConsumerCallbackMap{
		"default": callback,
	}, options...)
}

// RunConsumers runs the provided consumers as an application. You can pass additional options to customize the way it is executed.
//
// RunConsumers requires all consumers to accept the same input model. Thus, it is intended to be used if you have multiple source
// queues you are reading from.
func RunConsumers[M any](consumers stream.ConsumerCallbackMap[M], options ...Option) {
	factory := stream.NewConsumerFactory(consumers)

	options = append(options, WithModuleMultiFactory(factory))
	options = append(options, WithExecBackoffInfinite)

	Run(options...)
}

// RunUntypedConsumers runs the provided untyped consumers as an application. You can pass additional options to customize the way it is executed.
//
// Prefer using RunConsumers if all consumers share the same input type or use stream.EraseConsumerCallbackFactoryTypes to convert typed consumers
// to untyped consumers. If you are running distinct consumers in the same application, it might be a good idea to split the application into multiple
// applications if possible.
func RunUntypedConsumers(consumers stream.UntypedConsumerCallbackMap, options ...Option) {
	factory := stream.NewUntypedConsumerFactory(consumers)

	options = append(options, WithModuleMultiFactory(factory))
	options = append(options, WithExecBackoffInfinite)

	Run(options...)
}

func RunMdlSubscriber(transformers mdlsub.TransformerMapTypeVersionFactories, options ...Option) {
	subs := mdlsub.NewSubscriberFactory(transformers)

	options = append(options, WithModuleMultiFactory(subs))
	options = append(options, WithExecBackoffInfinite)

	Run(options...)
}
