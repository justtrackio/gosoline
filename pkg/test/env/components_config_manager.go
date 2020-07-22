package env

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"sync"
)

type ComponentsConfigManager struct {
	lck    sync.Mutex
	config cfg.GosoConf
}

func NewComponentsConfigManager(config cfg.GosoConf) *ComponentsConfigManager {
	return &ComponentsConfigManager{
		config: config,
	}
}

func (m *ComponentsConfigManager) GetAllSettings() ([]ComponentBaseSettingsAware, error) {
	allSettings := make([]ComponentBaseSettingsAware, 0)

	for i, configured := range m.List() {
		factory, ok := componentFactories[configured.Type]

		if !ok {
			return nil, fmt.Errorf("there is no component of type %s available", configured.Type)
		}

		settings := factory.GetSettingsSchema()

		key := fmt.Sprintf("test.components[%d]", i)
		m.config.UnmarshalKey(key, settings)

		allSettings = append(allSettings, settings)
	}

	return allSettings, nil
}

func (m *ComponentsConfigManager) List() []ComponentBaseSettings {
	settings := make([]ComponentBaseSettings, 0)

	if !m.config.IsSet("test.components") {
		return settings
	}

	m.config.UnmarshalKey("test.components", &settings)

	return settings
}

func (m *ComponentsConfigManager) Has(typ string, name string) bool {
	configured := m.List()

	for _, c := range configured {
		if typ != c.Type {
			continue
		}

		if name != c.Name {
			continue
		}

		return true
	}

	return false
}

func (m *ComponentsConfigManager) HasType(typ string) bool {
	configured := m.List()

	for _, c := range configured {
		if typ != c.Type {
			continue
		}

		return true
	}

	return false
}

func (m *ComponentsConfigManager) Add(settings interface{}) error {
	m.lck.Lock()
	defer m.lck.Unlock()

	componentSettings := settings.(ComponentBaseSettingsAware)

	if m.Has(componentSettings.GetName(), componentSettings.GetType()) {
		return nil
	}

	configured := m.List()
	key := fmt.Sprintf("test.components[%d]", len(configured))
	option := cfg.WithConfigSetting(key, componentSettings)

	if err := m.config.Option(option); err != nil {
		return fmt.Errorf("can not apply option onto config: %w", err)
	}

	return nil
}
