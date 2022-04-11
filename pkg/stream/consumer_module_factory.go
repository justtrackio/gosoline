package stream

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
)

type ConsumerCallbackMap[T comparable] map[string]ConsumerCallbackFactory[T]

func NewConsumerFactory[T comparable](callbacks ConsumerCallbackMap[T]) kernel.MultiModuleFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (map[string]kernel.ModuleFactory, error) {
		return ConsumerFactory(callbacks)
	}
}

func ConsumerFactory[T comparable](callbacks ConsumerCallbackMap[T]) (map[string]kernel.ModuleFactory, error) {
	modules := make(map[string]kernel.ModuleFactory)

	for name, callback := range callbacks {
		moduleName := fmt.Sprintf("consumer-%s", name)
		consumer := NewConsumer(name, callback)

		modules[moduleName] = consumer
	}

	return modules, nil
}
