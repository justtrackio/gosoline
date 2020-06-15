package env

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
)

func init() {
	componentFactories[componentSqs] = new(sqsFactory)
}

const componentSqs = "sqs"

type sqsSettings struct {
	ComponentBaseSettings
	ComponentContainerSettings
	Port int `cfg:"port" default:"0"`
}

type sqsFactory struct {
}

func (f *sqsFactory) Detect(config cfg.Config, manager *ComponentsConfigManager) error {
	if !config.IsSet("aws_sqs_endpoint") {
		return nil
	}

	if manager.HasType(componentSqs) {
		return nil
	}

	if err := manager.Add(componentSqs, "default"); err != nil {
		return fmt.Errorf("can not add default sqs component: %w", err)
	}

	return nil
}

func (f *sqsFactory) GetSettingsSchema() ComponentBaseSettingsAware {
	return &sqsSettings{}
}

func (f *sqsFactory) ConfigureContainer(settings interface{}) *containerConfig {
	s := settings.(*sqsSettings)

	return &containerConfig{
		Repository: "localstack/localstack",
		Tag:        "0.10.8",
		Env: []string{
			"SERVICES=sqs",
		},
		PortBindings: portBindings{
			"4576/tcp": s.Port,
			"8080/tcp": 0,
		},
		ExpireAfter: s.ExpireAfter,
	}
}

func (f *sqsFactory) HealthCheck(_ interface{}) ComponentHealthCheck {
	return localstackHealthCheck("sqs")
}

func (f *sqsFactory) Component(config cfg.Config, logger mon.Logger, container *container, settings interface{}) (Component, error) {
	component := &sqsComponent{
		binding: container.bindings["4576/tcp"],
	}

	return component, nil
}
