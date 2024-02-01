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

func RunConsumer(callback stream.ConsumerCallbackFactory, options ...Option) {
	RunConsumers(stream.ConsumerCallbackMap{
		"default": callback,
	}, options...)
}

func RunConsumers(consumers stream.ConsumerCallbackMap, options ...Option) {
	factory := stream.NewConsumerFactory(consumers)

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
