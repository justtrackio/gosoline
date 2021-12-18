package env

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

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
	localstackServiceS3         = "s3"
	localstackServiceSns        = "sns"
	localstackServiceSqs        = "sqs"
)

type localstackSettings struct {
	ComponentBaseSettings
	ComponentContainerSettings
	Port     int      `cfg:"port" default:"0"`
	Region   string   `cfg:"region" default:"eu-central-1"`
	Services []string `cfg:"services"`
}

type localstackFactory struct{}

func (f *localstackFactory) Detect(config cfg.Config, manager *ComponentsConfigManager) error {
	if manager.HasType(ComponentLocalstack) {
		return nil
	}

	if !manager.ShouldAutoDetect(ComponentLocalstack) {
		return nil
	}

	services := make([]string, 0)

	if config.IsSet("cloud.aws.cloudwatch") {
		services = append(services, localstackServiceCloudWatch)
	}

	if config.IsSet("aws_s3_endpoint") {
		services = append(services, localstackServiceS3)
	}

	if config.IsSet("cloud.aws.sns") {
		services = append(services, localstackServiceSns)
	}

	if config.IsSet("cloud.aws.sqs") {
		services = append(services, localstackServiceSqs)
	}

	if len(services) == 0 {
		return nil
	}

	settings := &localstackSettings{}
	config.UnmarshalDefaults(settings)

	settings.Type = ComponentLocalstack
	settings.Services = services

	if err := manager.Add(settings); err != nil {
		return fmt.Errorf("can not add default localstack component: %w", err)
	}

	return nil
}

func (f *localstackFactory) GetSettingsSchema() ComponentBaseSettingsAware {
	return &localstackSettings{}
}

func (f *localstackFactory) DescribeContainers(settings interface{}) componentContainerDescriptions {
	return componentContainerDescriptions{
		"main": {
			containerConfig: f.configureContainer(settings),
			healthCheck:     f.healthCheck(settings),
		},
	}
}

func (f *localstackFactory) configureContainer(settings interface{}) *containerConfig {
	s := settings.(*localstackSettings)
	services := strings.Join(s.Services, ",")

	return &containerConfig{
		Repository: "localstack/localstack",
		Tag:        "0.13.1",
		Env: []string{
			fmt.Sprintf("SERVICES=%s", services),
			fmt.Sprintf("DEFAULT_REGION=%s", s.Region),
			"EAGER_SERVICE_LOADING=1",
		},
		PortBindings: portBindings{
			"4566/tcp": s.Port,
		},
		ExpireAfter: s.ExpireAfter,
	}
}

func (f *localstackFactory) healthCheck(settings interface{}) ComponentHealthCheck {
	s := settings.(*localstackSettings)

	return func(container *container) error {
		binding := container.bindings["4566/tcp"]
		url := fmt.Sprintf("http://%s:%s/health", binding.host, binding.port)

		var err error
		var resp *http.Response
		var body []byte
		status := make(map[string]map[string]string)

		if resp, err = http.Get(url); err != nil {
			return err
		}

		if body, err = ioutil.ReadAll(resp.Body); err != nil {
			return err
		}

		if err = json.Unmarshal(body, &status); err != nil {
			return err
		}

		if _, ok := status["services"]; !ok {
			return fmt.Errorf("no localstack services up yet")
		}

		for _, service := range s.Services {
			if _, ok := status["services"][service]; !ok {
				return fmt.Errorf("%s service is not up yet", service)
			}

			if status["services"][service] != "running" {
				return fmt.Errorf("%s service is in %s state", service, status["services"][service])
			}
		}

		return nil
	}
}

func (f *localstackFactory) Component(_ cfg.Config, _ log.Logger, containers map[string]*container, settings interface{}) (Component, error) {
	s := settings.(*localstackSettings)

	component := &localstackComponent{
		services: s.Services,
		binding:  containers["main"].bindings["4566/tcp"],
		region:   s.Region,
	}

	return component, nil
}
