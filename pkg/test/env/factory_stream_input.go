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
	if has, err := manager.HasType(componentStreamInput); err != nil {
		return fmt.Errorf("failed to check if component exists: %w", err)
	} else if has {
		return nil
	}

	inputs, err := config.GetStringMap("stream.input", map[string]any{})
	if err != nil {
		return fmt.Errorf("can not get stream inputs: %w", err)
	}

	for inputName := range inputs {
		settings := &streamInputSettings{}
		if err := config.UnmarshalDefaults(settings); err != nil {
			return fmt.Errorf("could not unmarshal defaults for input %s: %w", inputName, err)
		}

		inMemoryOverride, err := config.GetBool(fmt.Sprintf("stream.input.%s.in_memory_override", inputName), settings.InMemoryOverride)
		if err != nil {
			return fmt.Errorf("could not get stream.input.%s.in_memory_override from config: %w", inputName, err)
		}

		settings.Name = inputName
		settings.Type = componentStreamInput
		settings.InMemoryOverride = inMemoryOverride

		if err := manager.Add(settings); err != nil {
			return fmt.Errorf("could not add input %s: %w", inputName, err)
		}
	}

	return nil
}

func (f streamInputFactory) GetSettingsSchema() ComponentBaseSettingsAware {
	return &streamInputSettings{}
}

func (f streamInputFactory) DescribeContainers(settings any) componentContainerDescriptions {
	return nil
}

func (f streamInputFactory) Component(_ cfg.Config, _ log.Logger, _ map[string]*container, settings any) (Component, error) {
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
