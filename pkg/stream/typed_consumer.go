package stream

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
)

//go:generate mockery --name TypedConsumerCallback
type TypedConsumerCallback[M any] interface {
	Consume(ctx context.Context, model M, attributes map[string]string) (bool, error)
}

type TypedConsumerCallbackFactory[M any] func(ctx context.Context, config cfg.Config, logger log.Logger) (TypedConsumerCallback[M], error)

type untypedConsumerCallback[M any] struct {
	consumerCallback TypedConsumerCallback[M]
}

var (
	_ InitializeableCallback   = untypedConsumerCallback[any]{}
	_ RunnableConsumerCallback = untypedConsumerCallback[any]{}
)

func NewTypedConsumer[M any](name string, callbackFactory TypedConsumerCallbackFactory[M]) kernel.ModuleFactory {
	return func(ctx context.Context, conf cfg.Config, logger log.Logger) (kernel.Module, error) {
		return NewConsumer(name, EraseConsumerCallbackFactoryTypes(callbackFactory))(ctx, conf, logger)
	}
}

func EraseConsumerCallbackFactoryTypes[M any](callbackFactory TypedConsumerCallbackFactory[M]) ConsumerCallbackFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (ConsumerCallback, error) {
		callback, err := callbackFactory(ctx, config, logger)
		if err != nil {
			return nil, err
		}

		return EraseConsumerCallbackTypes(callback), nil
	}
}

func EraseConsumerCallbackTypes[M any](consumerCallback TypedConsumerCallback[M]) ConsumerCallback {
	return untypedConsumerCallback[M]{
		consumerCallback: consumerCallback,
	}
}

func (u untypedConsumerCallback[M]) GetModel(_ map[string]string) any {
	return new(M)
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
