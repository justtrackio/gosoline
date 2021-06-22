package stream

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/kernel"
	"github.com/applike/gosoline/pkg/log"
)

type ConsumerCallbackMap map[string]ConsumerCallbackFactory

func NewConsumerFactory(callbacks ConsumerCallbackMap) kernel.MultiModuleFactory {
	return func(config cfg.Config, logger log.Logger) (map[string]kernel.ModuleFactory, error) {
		return ConsumerFactory(callbacks)
	}
}

func ConsumerFactory(callbacks ConsumerCallbackMap) (map[string]kernel.ModuleFactory, error) {
	modules := make(map[string]kernel.ModuleFactory)

	for name, callback := range callbacks {
		moduleName := fmt.Sprintf("consumer-%s", name)
		consumer := NewConsumer(name, callback)

		modules[moduleName] = consumer
	}

	return modules, nil
}
