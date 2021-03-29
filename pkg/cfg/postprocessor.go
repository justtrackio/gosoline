package cfg

import (
	"fmt"
	"sort"
)

type PostProcessor func(cfg GosoConf) (bool, error)

type postProcessorEntity struct {
	name      string
	processor PostProcessor
}

var postProcessorPriorities []int
var postProcessorEntities = map[int][]postProcessorEntity{}

func AddPostProcessor(priority int, name string, processor PostProcessor) {
	if _, ok := postProcessorEntities[priority]; !ok {
		postProcessorPriorities = append(postProcessorPriorities, priority)
		postProcessorEntities[priority] = make([]postProcessorEntity, 0)
	}

	entity := postProcessorEntity{
		name:      name,
		processor: processor,
	}

	postProcessorEntities[priority] = append(postProcessorEntities[priority], entity)
}

func ApplyPostProcessors(config GosoConf, logger Logger) error {
	sort.Ints(postProcessorPriorities)

	var err error
	var applied bool

	for i := len(postProcessorPriorities) - 1; i >= 0; i-- {
		priority := postProcessorPriorities[i]

		for _, entity := range postProcessorEntities[priority] {
			processor := entity.processor

			if applied, err = processor(config); err != nil {
				return fmt.Errorf("can not apply post processor '%s' on config: %w", entity.name, err)
			}

			if !applied {
				continue
			}

			logger.Info("applied priority %d config post processor '%s'", priority, entity.name)
		}
	}

	return nil
}
