package env

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
)

func init() {
	componentFactories[componentSns] = new(snsFactory)
}

const componentSns = "sns"

type snsSettings struct {
	ComponentBaseSettings
	ComponentContainerSettings
	Port int `cfg:"port" default:"0"`
}

type snsFactory struct {
}

func (f *snsFactory) Detect(config cfg.Config, manager *ComponentsConfigManager) error {
	if !config.IsSet("aws_sns_endpoint") {
		return nil
	}

	if manager.HasType(componentSns) {
		return nil
	}

	if err := manager.Add(componentSns, "default"); err != nil {
		return fmt.Errorf("can not add default sns component: %w", err)
	}

	return nil
}

func (f *snsFactory) GetSettingsSchema() ComponentBaseSettingsAware {
	return &snsSettings{}
}

func (f *snsFactory) ConfigureContainer(settings interface{}) *containerConfig {
	s := settings.(*snsSettings)

	return &containerConfig{
		Repository: "localstack/localstack",
		Tag:        "0.10.8",
		Env: []string{
			"SERVICES=sns",
		},
		PortBindings: portBindings{
			"4575/tcp": s.Port,
			"8080/tcp": 0,
		},
		ExpireAfter: s.ExpireAfter,
	}
}

func (f *snsFactory) HealthCheck(_ interface{}) ComponentHealthCheck {
	return localstackHealthCheck("sns")
}

func (f *snsFactory) Component(config cfg.Config, logger mon.Logger, container *container, _ interface{}) (Component, error) {
	component := &snsComponent{
		binding: container.bindings["4575/tcp"],
	}

	return component, nil
}
