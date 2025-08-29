package env

import (
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/stream"
)

func init() {
	componentFactories[componentStreamOutput] = new(streamOutputFactory)
}

const componentStreamOutput = "streamOutput"

type streamOutputSettings struct {
	ComponentBaseSettings
	InMemoryOverride bool `cfg:"in_memory_override" default:"true"`
}

type streamOutputFactory struct{}

func (f *streamOutputFactory) Detect(config cfg.Config, manager *ComponentsConfigManager) error {
	outputs, err := config.GetStringMap("stream.output", map[string]any{})
	if err != nil {
		return fmt.Errorf("can not get stream outputs: %w", err)
	}

	for outputName := range outputs {
		settings := &streamOutputSettings{}
		if err := config.UnmarshalDefaults(settings); err != nil {
			return fmt.Errorf("could not unmarshal defaults for output %s: %w", outputName, err)
		}

		inMemoryOverride, err := config.GetBool(fmt.Sprintf("stream.output.%s.in_memory_override", outputName), settings.InMemoryOverride)
		if err != nil {
			return fmt.Errorf("could not get stream.output.%s.in_memory_override from config: %w", outputName, err)
		}

		settings.Name = outputName
		settings.Type = componentStreamOutput
		settings.InMemoryOverride = inMemoryOverride

		if err := manager.Add(settings); err != nil {
			return fmt.Errorf("could not add output %s: %w", outputName, err)
		}
	}

	return nil
}

func (f streamOutputFactory) GetSettingsSchema() ComponentBaseSettingsAware {
	return &streamOutputSettings{}
}

func (f streamOutputFactory) DescribeContainers(settings any) componentContainerDescriptions {
	return nil
}

func (f streamOutputFactory) Component(_ cfg.Config, _ log.Logger, _ map[string]*container, settings any) (Component, error) {
	s := settings.(*streamOutputSettings)

	component := &streamOutputComponent{
		name:             s.Name,
		output:           stream.ProvideInMemoryOutput(s.Name),
		encoder:          stream.NewMessageEncoder(&stream.MessageEncoderSettings{}),
		inMemoryOverride: s.InMemoryOverride,
	}

	return component, nil
}
