package env

import (
	"fmt"
	"io"
	"net/http"

	toxiproxy "github.com/Shopify/toxiproxy/v2/client"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/ddb"
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
	Port             int    `cfg:"port"              default:"0"`
	Region           string `cfg:"region"            default:"eu-central-1"`
	ToxiproxyEnabled bool   `cfg:"toxiproxy_enabled" default:"false"`
}

type localstackFactory struct {
	toxiproxyFactory toxiproxyFactory
}

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
	s := settings.(*localstackSettings)

	descriptions := ComponentContainerDescriptions{
		"main": {
			ContainerConfig: f.configureContainer(settings),
			HealthCheck:     f.healthCheck(settings),
		},
	}

	if s.ToxiproxyEnabled {
		descriptions["toxiproxy"] = f.toxiproxyFactory.describeContainer()
	}

	return descriptions
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

func (f *localstackFactory) Component(config cfg.Config, logger log.Logger, containers map[string]*Container, settings any) (Component, error) {
	var err error
	var ddbNamingSettings *ddb.TableNamingSettings
	var proxy *toxiproxy.Proxy

	s := settings.(*localstackSettings)
	endpoint := containers["main"].bindings["main"].getAddress()

	if ddbNamingSettings, err = ddb.GetTableNamingSettings(config, s.Name); err != nil {
		return nil, fmt.Errorf("can not get table naming settings for ddb component: %w", err)
	}

	if s.ToxiproxyEnabled {
		toxiproxyClient := f.toxiproxyFactory.client(containers["toxiproxy"])

		if proxy, err = toxiproxyClient.CreateProxy("ddb", ":56248", endpoint); err != nil {
			return nil, fmt.Errorf("can not create toxiproxy proxy for ddb component: %w", err)
		}

		endpoint = containers["toxiproxy"].bindings["main"].getAddress()
	}

	component := &localstackComponent{
		config:            config,
		logger:            logger,
		endpointAddress:   endpoint,
		region:            s.Region,
		ddbNamingSettings: ddbNamingSettings,
		toxiproxy:         proxy,
	}

	return component, nil
}
