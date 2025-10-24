package env

import (
	"fmt"
	"io"
	"net/http"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/encoding/json"
	"github.com/justtrackio/gosoline/pkg/log"
)

func init() {
	componentFactories[ComponentLocalstack] = new(localstackFactory)
}

const (
	ComponentLocalstack         = "localstack"
	localstackServiceCloudWatch = "cloudwatch"
	localstackServicesKey       = "services"
	localstackServiceS3         = "s3"
	localstackServiceSns        = "sns"
	localstackServiceSqs        = "sqs"
)

type localstackSettings struct {
	ComponentBaseSettings
	ComponentContainerSettings
	Port   int    `cfg:"port" default:"0"`
	Region string `cfg:"region" default:"eu-central-1"`
}

type localstackFactory struct{}

func (f *localstackFactory) Detect(config cfg.Config, manager *ComponentsConfigManager) error {
	if has, err := manager.HasType(ComponentLocalstack); err != nil {
		return fmt.Errorf("failed to check if component exists: %w", err)
	} else if has {
		return nil
	}

	if !manager.ShouldAutoDetect(ComponentLocalstack) {
		return nil
	}

	if !config.IsSet("cloud.aws") {
		return nil
	}

	settings := &localstackSettings{}
	if err := UnmarshalSettings(config, settings, ComponentLocalstack, "default"); err != nil {
		return fmt.Errorf("can not detect localstack settings: %w", err)
	}
	settings.Type = ComponentLocalstack

	if err := manager.Add(settings); err != nil {
		return fmt.Errorf("can not add default localstack component: %w", err)
	}

	return nil
}

func (f *localstackFactory) GetSettingsSchema() ComponentBaseSettingsAware {
	return &localstackSettings{}
}

func (f *localstackFactory) DescribeContainers(settings any) ComponentContainerDescriptions {
	return ComponentContainerDescriptions{
		"main": {
			ContainerConfig: f.configureContainer(settings),
			HealthCheck:     f.healthCheck(settings),
		},
	}
}

func (f *localstackFactory) configureContainer(settings any) *ContainerConfig {
	s := settings.(*localstackSettings)

	return &ContainerConfig{
		Auth:       s.Image.Auth,
		Repository: s.Image.Repository,
		Tag:        s.Image.Tag,
		PortBindings: PortBindings{
			"main": {
				ContainerPort: 4566,
				HostPort:      s.Port,
				Protocol:      "tcp",
			},
		},
	}
}

func (f *localstackFactory) healthCheck(settings any) ComponentHealthCheck {
	return func(container *Container) error {
		binding := container.bindings["main"]
		url := fmt.Sprintf("http://%s:%s/_localstack/health", binding.host, binding.port)

		var err error
		var resp *http.Response
		var body []byte
		status := make(map[string]any)

		if resp, err = http.Get(url); err != nil {
			return err
		}

		if body, err = io.ReadAll(resp.Body); err != nil {
			return err
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("status %d: %s", resp.StatusCode, body)
		}

		if err := json.Unmarshal(body, &status); err != nil {
			return err
		}

		if _, ok := status[localstackServicesKey]; !ok {
			return fmt.Errorf("no localstack services up yet")
		}

		return nil
	}
}

func (f *localstackFactory) Component(_ cfg.Config, _ log.Logger, containers map[string]*Container, settings any) (Component, error) {
	s := settings.(*localstackSettings)

	component := &localstackComponent{
		binding: containers["main"].bindings["main"],
		region:  s.Region,
	}

	return component, nil
}
