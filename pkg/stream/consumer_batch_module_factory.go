package stream

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
)

type (
	BatchConsumerCallbackMap[M any] map[string]BatchConsumerCallbackFactory[M]
	UntypedBatchConsumerCallbackMap map[string]UntypedBatchConsumerCallbackFactory
)

func NewBatchConsumerFactory[M any](callbacks BatchConsumerCallbackMap[M]) kernel.ModuleMultiFactory {
	untypedCallbacks := funk.MapValues(callbacks, EraseBatchConsumerCallbackFactoryTypes)

	return NewUntypedBatchConsumerFactory(untypedCallbacks)
}

func NewUntypedBatchConsumerFactory(callbacks UntypedBatchConsumerCallbackMap) kernel.ModuleMultiFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (map[string]kernel.ModuleFactory, error) {
		return BatchConsumerFactory(callbacks)
	}
}

func BatchConsumerFactory(callbacks UntypedBatchConsumerCallbackMap) (map[string]kernel.ModuleFactory, error) {
	modules := make(map[string]kernel.ModuleFactory)

	for name, callback := range callbacks {
		moduleName := fmt.Sprintf("consumer-%s", name)
		consumer := NewUntypedBatchConsumer(name, callback)

		modules[moduleName] = consumer
	}

	return modules, nil
}
