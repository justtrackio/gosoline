package stream

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
)

//go:generate go run github.com/vektra/mockery/v2 --name BatchConsumerCallback
type BatchConsumerCallback[M any] interface {
	Consume(ctx context.Context, models []M, attributes []map[string]string) ([]bool, error)
}

//go:generate go run github.com/vektra/mockery/v2 --name RunnableBatchConsumerCallback
type RunnableBatchConsumerCallback[M any] interface {
	BatchConsumerCallback[M]
	RunnableCallback
}

type BatchConsumerCallbackFactory[M any] func(ctx context.Context, config cfg.Config, logger log.Logger) (BatchConsumerCallback[M], error)

type untypedBatchConsumerCallback[M any] struct {
	consumerCallback BatchConsumerCallback[M]
}

var (
	_ InitializeableCallback               = untypedBatchConsumerCallback[any]{}
	_ RunnableUntypedBatchConsumerCallback = untypedBatchConsumerCallback[any]{}
	_ SchemaSettingsAwareCallback          = untypedBatchConsumerCallback[any]{}
)

func NewTypedBatchConsumer[M any](name string, callbackFactory BatchConsumerCallbackFactory[M]) kernel.ModuleFactory {
	return func(ctx context.Context, conf cfg.Config, logger log.Logger) (kernel.Module, error) {
		return NewUntypedBatchConsumer(name, EraseBatchConsumerCallbackFactoryTypes(callbackFactory))(ctx, conf, logger)
	}
}

func EraseBatchConsumerCallbackFactoryTypes[M any](callbackFactory BatchConsumerCallbackFactory[M]) UntypedBatchConsumerCallbackFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (UntypedBatchConsumerCallback, error) {
		callback, err := callbackFactory(ctx, config, logger)
		if err != nil {
			return nil, err
		}

		return EraseBatchConsumerCallbackTypes(callback), nil
	}
}

func EraseBatchConsumerCallbackTypes[M any](consumerCallback BatchConsumerCallback[M]) UntypedBatchConsumerCallback {
	return untypedBatchConsumerCallback[M]{
		consumerCallback: consumerCallback,
	}
}

func (u untypedBatchConsumerCallback[M]) GetModel(_ map[string]string) any {
	return new(M)
}

func (u untypedBatchConsumerCallback[M]) Consume(ctx context.Context, models []any, attributes []map[string]string) ([]bool, error) {
	inputs := funk.Map(models, func(model any) M {
		return *model.(*M)
	})

	return u.consumerCallback.Consume(ctx, inputs, attributes)
}

func (u untypedBatchConsumerCallback[M]) Init(ctx context.Context) error {
	if initializeable, ok := u.consumerCallback.(InitializeableCallback); ok {
		return initializeable.Init(ctx)
	}

	return nil
}

func (u untypedBatchConsumerCallback[M]) Run(ctx context.Context) error {
	if runnable, ok := u.consumerCallback.(RunnableCallback); ok {
		return runnable.Run(ctx)
	}

	return nil
}

func (u untypedBatchConsumerCallback[M]) GetSchemaSettings() (*SchemaSettings, error) {
	if schemaAware, ok := u.consumerCallback.(SchemaSettingsAwareCallback); ok {
		return schemaAware.GetSchemaSettings()
	}

	return nil, nil
}
