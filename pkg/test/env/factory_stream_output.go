package env

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/stream"
)

func init() {
	componentFactories[componentStreamOutput] = new(streamOutputFactory)
}

const componentStreamOutput = "streamOutput"

type streamOutputSettings struct {
	ComponentBaseSettings
}

type streamOutputFactory struct {
}

func (f *streamOutputFactory) Detect(config cfg.Config, manager *ComponentsConfigManager) error {
	outputs := config.GetStringMap("stream.output", map[string]interface{}{})

	for outputName := range outputs {
		settings := &streamOutputSettings{}
		config.UnmarshalDefaults(settings)

		settings.Name = outputName
		settings.Type = componentStreamOutput

		if err := manager.Add(settings); err != nil {
			return fmt.Errorf("could not add output %s: %w", outputName, err)
		}
	}

	return nil
}

func (f streamOutputFactory) GetSettingsSchema() ComponentBaseSettingsAware {
	return &streamOutputSettings{}
}

func (f streamOutputFactory) ConfigureContainer(_ interface{}) *containerConfig {
	return nil
}

func (f streamOutputFactory) HealthCheck(_ interface{}) ComponentHealthCheck {
	return nil
}

func (f streamOutputFactory) Component(_ cfg.Config, _ mon.Logger, _ *container, settings interface{}) (Component, error) {
	s := settings.(*streamOutputSettings)

	component := &streamOutputComponent{
		name:   s.Name,
		output: stream.ProvideInMemoryOutput(s.Name),
	}

	return component, nil
}
