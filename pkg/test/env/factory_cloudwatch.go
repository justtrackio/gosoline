package env

import "github.com/applike/gosoline/pkg/cfg"

const componentCloudwatch = "cloudwatch"

type cloudwatchSettings struct {
	ComponentBaseSettings
	ComponentContainerSettings
	Port int `cfg:"port" default:"0"`
}

type cloudwatchFactory struct {
}

func (f *cloudwatchFactory) Detect(config cfg.Config, manager *ComponentsConfigManager) error {
	return nil
}

func (f *cloudwatchFactory) GetSettingsSchema() ComponentBaseSettingsAware {
	return &cloudwatchSettings{}
}

func (f *cloudwatchFactory) ConfigureContainer(settings interface{}) *containerConfig {
	s := settings.(*cloudwatchSettings)

	return &containerConfig{
		Repository: "localstack/localstack",
		Tag:        "0.10.8",
		Env: []string{
			"SERVICES=cloudwatch",
		},
		PortBindings: portBindings{
			"4582/tcp": s.Port,
			"8080/tcp": 0,
		},
		ExpireAfter: s.ExpireAfter,
	}
}

func (f *cloudwatchFactory) HealthCheck(settings interface{}) ComponentHealthCheck {
	return nil
}

func (f *cloudwatchFactory) Component(settings interface{}, container *container) (Component, error) {
	return nil, nil
}
