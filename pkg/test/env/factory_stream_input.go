package env

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/stream"
)

const componentStreamInput = "streamInput"

type streamInputSettings struct {
	ComponentBaseSettings
}

type streamInputFactory struct {
}

func (f *streamInputFactory) Detect(config cfg.Config, manager *ComponentsConfigManager) error {
	inputs := config.GetStringMap("stream.input", map[string]interface{}{})

	for inputName := range inputs {
		if err := manager.Add(componentStreamInput, inputName); err != nil {
			return fmt.Errorf("could not add input %s: %w", inputName, err)
		}
	}

	return nil
}

func (f streamInputFactory) GetSettingsSchema() ComponentBaseSettingsAware {
	return &streamInputSettings{}
}

func (f streamInputFactory) ConfigureContainer(_ interface{}) *containerConfig {
	return nil
}

func (f streamInputFactory) HealthCheck(_ interface{}) ComponentHealthCheck {
	return nil
}

func (f streamInputFactory) Component(settings interface{}, _ *container) (Component, error) {
	s := settings.(*streamInputSettings)

	component := &streamInputComponent{
		name: s.Name,
		input: stream.ProvideInMemoryInput(s.Name, &stream.InMemorySettings{
			Size: 10,
		}),
	}

	return component, nil
}
