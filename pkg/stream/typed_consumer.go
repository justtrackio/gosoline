package stream

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
)

//go:generate go run github.com/vektra/mockery/v2 --name ConsumerCallback
type ConsumerCallback[M any] interface {
	Consume(ctx context.Context, model M, attributes map[string]string) (bool, error)
}

//go:generate go run github.com/vektra/mockery/v2 --name RunnableConsumerCallback
type RunnableConsumerCallback[M any] interface {
	ConsumerCallback[M]
	RunnableCallback
}

type ConsumerCallbackFactory[M any] func(ctx context.Context, config cfg.Config, logger log.Logger) (ConsumerCallback[M], error)

type untypedConsumerCallback[M any] struct {
	consumerCallback ConsumerCallback[M]
}

var (
	_ InitializeableCallback          = untypedConsumerCallback[any]{}
	_ RunnableUntypedConsumerCallback = untypedConsumerCallback[any]{}
	_ SchemaSettingsAwareCallback     = untypedConsumerCallback[any]{}
)

func NewConsumer[M any](name string, callbackFactory ConsumerCallbackFactory[M]) kernel.ModuleFactory {
	return func(ctx context.Context, conf cfg.Config, logger log.Logger) (kernel.Module, error) {
		return NewUntypedConsumer(name, EraseConsumerCallbackFactoryTypes(callbackFactory))(ctx, conf, logger)
	}
}

func EraseConsumerCallbackFactoryTypes[M any](callbackFactory ConsumerCallbackFactory[M]) UntypedConsumerCallbackFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (UntypedConsumerCallback, error) {
		callback, err := callbackFactory(ctx, config, logger)
		if err != nil {
			return nil, err
		}

		return EraseConsumerCallbackTypes(callback), nil
	}
}

func EraseConsumerCallbackTypes[M any](consumerCallback ConsumerCallback[M]) UntypedConsumerCallback {
	return untypedConsumerCallback[M]{
		consumerCallback: consumerCallback,
	}
}

func (u untypedConsumerCallback[M]) GetModel(_ map[string]string) (any, error) {
	return new(M), nil
}

func (u untypedConsumerCallback[M]) Consume(ctx context.Context, model any, attributes map[string]string) (bool, error) {
	input := model.(*M)

	return u.consumerCallback.Consume(ctx, *input, attributes)
}

func (u untypedConsumerCallback[M]) Init(ctx context.Context) error {
	if initializeable, ok := u.consumerCallback.(InitializeableCallback); ok {
		return initializeable.Init(ctx)
	}

	return nil
}

func (u untypedConsumerCallback[M]) Run(ctx context.Context) error {
	if runnable, ok := u.consumerCallback.(RunnableCallback); ok {
		return runnable.Run(ctx)
	}

	return nil
}

func (u untypedConsumerCallback[M]) GetSchemaSettings() (*SchemaSettings, error) {
	if schemaAware, ok := u.consumerCallback.(SchemaSettingsAwareCallback); ok {
		return schemaAware.GetSchemaSettings()
	}

	return nil, nil
}
