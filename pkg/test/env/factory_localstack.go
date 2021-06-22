package env

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/encoding/json"
	"github.com/applike/gosoline/pkg/log"
	"io/ioutil"
	"net/http"
	"strings"
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

type localstackFactory struct {
}

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

	if config.IsSet("aws_sns_endpoint") {
		services = append(services, localstackServiceSns)
	}

	if config.IsSet("aws_sqs_endpoint") {
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
		Tag:        "0.12.5",
		Env: []string{
			fmt.Sprintf("SERVICES=%s", services),
			fmt.Sprintf("DEFAULT_REGION=%s", s.Region),
		},
		PortBindings: portBindings{
			"4566/tcp": s.Port,
			"8080/tcp": 0,
		},
		ExpireAfter: s.ExpireAfter,
	}
}

func (f *localstackFactory) healthCheck(settings interface{}) ComponentHealthCheck {
	s := settings.(*localstackSettings)

	return func(container *container) error {
		binding := container.bindings["4566/tcp"]
		url := fmt.Sprintf("http://%s:%s/health?reload", binding.host, binding.port)

		var err error
		var resp *http.Response
		var body []byte
		var status = make(map[string]map[string]string)

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
			return fmt.Errorf("sns service is not up yet")
		}

		for _, service := range s.Services {
			if _, ok := status["services"][service]; !ok {
				return fmt.Errorf("%s service is not up yet", service)
			}

			if status["services"][service] != "running" {
				return fmt.Errorf("%s service is in %s state", service, status["services"]["sns"])
			}
		}

		return nil
	}
}

func (f *localstackFactory) Component(config cfg.Config, logger log.Logger, containers map[string]*container, settings interface{}) (Component, error) {
	s := settings.(*localstackSettings)

	component := &localstackComponent{
		services: s.Services,
		binding:  containers["main"].bindings["4566/tcp"],
		region:   s.Region,
	}

	return component, nil
}
