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
	ConsumerCallbackMap[M any] map[string]ConsumerCallbackFactory[M]
	UntypedConsumerCallbackMap map[string]UntypedConsumerCallbackFactory
)

func NewConsumerFactory[M any](callbacks ConsumerCallbackMap[M]) kernel.ModuleMultiFactory {
	untypedCallbacks := funk.MapValues(callbacks, EraseConsumerCallbackFactoryTypes)

	return NewUntypedConsumerFactory(untypedCallbacks)
}

func NewUntypedConsumerFactory(callbacks UntypedConsumerCallbackMap) kernel.ModuleMultiFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (map[string]kernel.ModuleFactory, error) {
		return ConsumerFactory(callbacks)
	}
}

func ConsumerFactory(callbacks UntypedConsumerCallbackMap) (map[string]kernel.ModuleFactory, error) {
	modules := make(map[string]kernel.ModuleFactory)

	for name, callback := range callbacks {
		moduleName := fmt.Sprintf("consumer-%s", name)
		consumer := NewUntypedConsumer(name, callback)

		modules[moduleName] = consumer
	}

	return modules, nil
}
