package env

import (
	"fmt"
	"slices"
	"sync"

	"github.com/justtrackio/gosoline/pkg/cfg"
)

type AutoDetectSettings struct {
	Enabled        bool     `cfg:"enabled" default:"true"`
	SkipComponents []string `cfg:"skip_components"`
}

type ComponentsConfigManager struct {
	lck    sync.Mutex
	config cfg.GosoConf
	detect *AutoDetectSettings
}

func NewComponentsConfigManager(config cfg.GosoConf) *ComponentsConfigManager {
	autoDetectSettings := &AutoDetectSettings{}
	config.UnmarshalKey("test.auto_detect", autoDetectSettings)

	return &ComponentsConfigManager{
		config: config,
		detect: autoDetectSettings,
	}
}

func (m *ComponentsConfigManager) ShouldAutoDetect(typ string) bool {
	return m.detect.Enabled && !slices.Contains(m.detect.SkipComponents, typ)
}

func (m *ComponentsConfigManager) GetAllSettings() ([]ComponentBaseSettingsAware, error) {
	allSettings := make([]ComponentBaseSettingsAware, 0)

	for typ, components := range m.List() {
		factory, ok := componentFactories[typ]

		if !ok {
			return nil, fmt.Errorf("there is no component of type %s available", typ)
		}

		for name := range components {
			settings := factory.GetSettingsSchema()
			UnmarshalSettings(m.config, settings, typ, name)
			allSettings = append(allSettings, settings)
		}
	}

	return allSettings, nil
}

func (m *ComponentsConfigManager) List() map[string]map[string]interface{} {
	settings := make(map[string]map[string]interface{}, 0)

	if !m.config.IsSet("test.components") {
		return settings
	}

	types := m.config.GetStringMap("test.components")
	for typ := range types {
		settings[typ] = make(map[string]interface{}, 0)

		names := m.config.GetStringMap(fmt.Sprintf("test.components.%s", typ))
		for name, value := range names {
			settings[typ][name] = value
		}
	}

	return settings
}

func (m *ComponentsConfigManager) Has(typ string, name string) bool {
	configured := m.List()

	if components, ok := configured[typ]; ok {
		if _, ok = components[name]; ok {
			return true
		}
	}

	return false
}

func (m *ComponentsConfigManager) HasType(typ string) bool {
	configured := m.List()

	if _, ok := configured[typ]; ok {
		return true
	}

	return false
}

func (m *ComponentsConfigManager) Add(settings interface{}) error {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println(settings)
		}
	}()

	m.lck.Lock()
	defer m.lck.Unlock()

	componentSettings, ok := settings.(ComponentBaseSettingsAware)

	if !ok {
		return fmt.Errorf("the component settings has to implement the interface ComponentBaseSettingsAware")
	}

	if m.Has(componentSettings.GetName(), componentSettings.GetType()) {
		return fmt.Errorf("component %s of type %s already exists", componentSettings.GetName(), componentSettings.GetType())
	}

	key := fmt.Sprintf("test.components.%s.%s", componentSettings.GetType(), componentSettings.GetName())
	option := cfg.WithConfigSetting(key, componentSettings)

	if err := m.config.Option(option); err != nil {
		return fmt.Errorf("can not apply option onto config: %w", err)
	}

	return nil
}

func UnmarshalSettings(config cfg.Config, settings interface{}, typ string, name string) {
	key := fmt.Sprintf("test.components.%s.%s", typ, name)
	defaultKey := fmt.Sprintf("test.defaults.images.%s", typ)

	defaults := []cfg.UnmarshalDefaults{
		cfg.UnmarshalWithDefaultForKey("type", typ),
		cfg.UnmarshalWithDefaultForKey("name", name),
		cfg.UnmarshalWithDefaultsFromKey(defaultKey, "image"),
	}

	config.UnmarshalKey(key, settings, defaults...)
}
