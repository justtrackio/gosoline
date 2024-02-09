package env

import (
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/stream"
)

func init() {
	componentFactories[componentStreamInput] = new(streamInputFactory)
}

const componentStreamInput = "streamInput"

type streamInputSettings struct {
	ComponentBaseSettings
	InMemoryOverride bool `cfg:"in_memory_override" default:"true"`
}

type streamInputFactory struct{}

func (f *streamInputFactory) Detect(config cfg.Config, manager *ComponentsConfigManager) error {
	if manager.HasType(componentStreamInput) {
		return nil
	}

	inputs := config.GetStringMap("stream.input", map[string]interface{}{})

	for inputName := range inputs {
		settings := &streamInputSettings{}
		config.UnmarshalDefaults(settings)

		settings.Name = inputName
		settings.Type = componentStreamInput

		if err := manager.Add(settings); err != nil {
			return fmt.Errorf("could not add input %s: %w", inputName, err)
		}
	}

	return nil
}

func (f streamInputFactory) GetSettingsSchema() ComponentBaseSettingsAware {
	return &streamInputSettings{}
}

func (f streamInputFactory) DescribeContainers(settings interface{}) componentContainerDescriptions {
	return nil
}

func (f streamInputFactory) Component(_ cfg.Config, _ log.Logger, _ map[string]*container, settings interface{}) (Component, error) {
	s := settings.(*streamInputSettings)

	component := &StreamInputComponent{
		name: s.Name,
		input: stream.ProvideInMemoryInput(s.Name, &stream.InMemorySettings{
			Size: 10,
		}),
		inMemoryOverride: s.InMemoryOverride,
	}

	return component, nil
}
