package stream

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/kernel"
	"github.com/applike/gosoline/pkg/mon"
)

type ConsumerCallbackMap map[string]ConsumerCallback

func NewConsumerFactory(callbacks ConsumerCallbackMap) kernel.ModuleFactory {
	return func(config cfg.Config, logger mon.Logger) (map[string]kernel.Module, error) {
		return ConsumerFactory(config, logger, callbacks)
	}
}

func ConsumerFactory(config cfg.Config, logger mon.Logger, callbacks ConsumerCallbackMap) (map[string]kernel.Module, error) {
	modules := make(map[string]kernel.Module)

	for name, callback := range callbacks {
		moduleName := fmt.Sprintf("consumer-%s", name)
		consumer := NewConsumer(name, callback)

		modules[moduleName] = consumer
	}

	return modules, nil
}
