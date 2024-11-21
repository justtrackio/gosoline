package stream

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
)

type BatchConsumerCallbackMap map[string]BatchConsumerCallbackFactory

func NewBatchConsumerFactory(callbacks BatchConsumerCallbackMap) kernel.ModuleMultiFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (map[string]kernel.ModuleFactory, error) {
		return BatchConsumerFactory(callbacks)
	}
}

func BatchConsumerFactory(callbacks BatchConsumerCallbackMap) (map[string]kernel.ModuleFactory, error) {
	modules := make(map[string]kernel.ModuleFactory)

	for name, callback := range callbacks {
		moduleName := fmt.Sprintf("batch-consumer-%s", name)
		consumer := NewBatchConsumer(name, callback)

		modules[moduleName] = consumer
	}

	return modules, nil
}
