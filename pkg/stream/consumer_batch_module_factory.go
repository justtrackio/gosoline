package stream

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
)

// BatchConsumerCallbackMap associates consumer names with typed batch factories.
type BatchConsumerCallbackMap[M any] map[string]BatchConsumerCallbackFactory[M]

// UntypedBatchConsumerCallbackMap associates names with untyped batch factories.
type UntypedBatchConsumerCallbackMap map[string]UntypedBatchConsumerCallbackFactory

// NewBatchConsumerFactory constructs modules for typed batch callbacks.
func NewBatchConsumerFactory[M any](callbacks BatchConsumerCallbackMap[M]) kernel.ModuleMultiFactory {
	return NewUntypedBatchConsumerFactory(funk.MapValues(callbacks, EraseBatchConsumerCallbackFactoryTypes))
}

// NewUntypedBatchConsumerFactory constructs modules for untyped batch callbacks.
func NewUntypedBatchConsumerFactory(callbacks UntypedBatchConsumerCallbackMap) kernel.ModuleMultiFactory {
	return func(context.Context, cfg.Config, log.Logger) (map[string]kernel.ModuleFactory, error) {
		return BatchConsumerFactory(callbacks)
	}
}

// BatchConsumerFactory names batch modules consistently with ordinary consumers.
func BatchConsumerFactory(callbacks UntypedBatchConsumerCallbackMap) (map[string]kernel.ModuleFactory, error) {
	modules := make(map[string]kernel.ModuleFactory, len(callbacks))
	for name, callback := range callbacks {
		modules[fmt.Sprintf("consumer-%s", name)] = NewUntypedBatchConsumer(name, callback)
	}

	return modules, nil
}

// BatchConsumerCallback processes models serially in size/time-triggered batches.
// The returned decisions control retries and retry-input acknowledgement;
// primary inputs acknowledge admission before business processing.
//
//go:generate go run github.com/vektra/mockery/v2 --name BatchConsumerCallback
type BatchConsumerCallback[M any] interface {
	Consume(context.Context, []M, []map[string]string) ([]bool, error)
}

// RunnableBatchConsumerCallback also runs alongside batch collection.
//
//go:generate go run github.com/vektra/mockery/v2 --name RunnableBatchConsumerCallback
type RunnableBatchConsumerCallback[M any] interface {
	BatchConsumerCallback[M]
	RunnableCallback
}

// BatchConsumerCallbackFactory constructs a typed batch callback.
type BatchConsumerCallbackFactory[M any] func(context.Context, cfg.Config, log.Logger) (BatchConsumerCallback[M], error)

type untypedBatchConsumerCallback[M any] struct {
	callback BatchConsumerCallback[M]
}

// NewBatchConsumer constructs a batch consumer with typed model decoding.
func NewBatchConsumer[M any](name string, factory BatchConsumerCallbackFactory[M]) kernel.ModuleFactory {
	return NewUntypedBatchConsumer(name, EraseBatchConsumerCallbackFactoryTypes(factory))
}

// EraseBatchConsumerCallbackFactoryTypes adapts a typed callback factory.
func EraseBatchConsumerCallbackFactoryTypes[M any](factory BatchConsumerCallbackFactory[M]) UntypedBatchConsumerCallbackFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (UntypedBatchConsumerCallback, error) {
		var err error
		var callback BatchConsumerCallback[M]

		if callback, err = factory(ctx, config, logger); err != nil {
			return nil, err
		}

		return EraseBatchConsumerCallbackTypes(callback), nil
	}
}

// EraseBatchConsumerCallbackTypes adapts a typed callback, including its optional
// initialization, runnable and schema-settings interfaces.
func EraseBatchConsumerCallbackTypes[M any](callback BatchConsumerCallback[M]) UntypedBatchConsumerCallback {
	return untypedBatchConsumerCallback[M]{callback: callback}
}

func (u untypedBatchConsumerCallback[M]) GetModel(map[string]string) (any, error) {
	return new(M), nil
}

func (u untypedBatchConsumerCallback[M]) Consume(ctx context.Context, models []any, attributes []map[string]string) ([]bool, error) {
	typed := make([]M, len(models))
	for i, model := range models {
		typed[i] = *model.(*M)
	}

	return u.callback.Consume(ctx, typed, attributes)
}

func (u untypedBatchConsumerCallback[M]) Init(ctx context.Context) error {
	if callback, ok := u.callback.(InitializeableCallback); ok {
		return callback.Init(ctx)
	}

	return nil
}

func (u untypedBatchConsumerCallback[M]) Run(ctx context.Context) error {
	if callback, ok := u.callback.(RunnableCallback); ok {
		return callback.Run(ctx)
	}

	return nil
}

func (u untypedBatchConsumerCallback[M]) GetSchemaSettings() (*SchemaSettings, error) {
	if callback, ok := u.callback.(SchemaSettingsAwareCallback); ok {
		return callback.GetSchemaSettings()
	}

	return nil, nil
}
