package env

import (
	"context"
	"fmt"

	toxiproxy "github.com/Shopify/toxiproxy/v2/client"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

func init() {
	componentFactories[ComponentFloci] = new(flociFactory)
}

// ComponentFloci identifies the shared Floci AWS test service.
const ComponentFloci = "floci"

type flociSettings struct {
	ComponentBaseSettings
	ComponentContainerSettings
	ContainerBindingSettings
	Region           string `cfg:"region"            default:"eu-central-1"`
	ToxiproxyEnabled bool   `cfg:"toxiproxy_enabled" default:"false"`
}

type flociFactory struct {
	toxiproxyFactory toxiproxyFactory
}

func (f *flociFactory) Detect(config cfg.Config, manager *ComponentsConfigManager) error {
	if has, err := manager.HasType(ComponentFloci); err != nil {
		return fmt.Errorf("failed to check if component exists: %w", err)
	} else if has {
		return nil
	}

	if !manager.ShouldAutoDetect(ComponentFloci) {
		return nil
	}

	if !config.IsSet("cloud.aws") {
		return nil
	}

	settings := &flociSettings{}
	if err := UnmarshalSettings(config, settings, ComponentFloci, "default"); err != nil {
		return fmt.Errorf("can not detect floci settings: %w", err)
	}
	settings.Type = ComponentFloci

	if err := manager.Add(settings); err != nil {
		return fmt.Errorf("can not add default floci component: %w", err)
	}

	return nil
}

func (f *flociFactory) GetSettingsSchema() ComponentBaseSettingsAware {
	return &flociSettings{}
}

func (f *flociFactory) DescribeContainers(settings any) ComponentContainerDescriptions {
	s := settings.(*flociSettings)

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

func (f *flociFactory) configureContainer(settings any) *ContainerConfig {
	s := settings.(*flociSettings)

	ports := PortBindings{"main": {ContainerPort: 4566, HostPort: s.Port, Protocol: "tcp"}}
	if s.isExternal() {
		return externalContainer(s.Host, ports)
	}

	return &ContainerConfig{
		Auth:         s.Image.Auth,
		Repository:   s.Image.Repository,
		Tag:          s.Image.Tag,
		PortBindings: ports,
	}
}

func (f *flociFactory) healthCheck(settings any) ComponentHealthCheck {
	return func(container *Container) error {
		component := &dynamoDbComponent{binding: container.bindings["main"], region: settings.(*flociSettings).Region}
		_, err := component.DdbClient().ListTables(context.Background(), &dynamodb.ListTablesInput{})

		return err
	}
}

func (f *flociFactory) Component(config cfg.Config, logger log.Logger, containers map[string]*Container, settings any) (Component, error) {
	var err error
	var proxy *toxiproxy.Proxy

	s := settings.(*flociSettings)
	clientContainer := containers["main"]
	endpoint := clientContainer.bindings["main"].getAddress()

	if s.ToxiproxyEnabled {
		toxiproxyClient := f.toxiproxyFactory.client(containers["toxiproxy"])

		// Use the internal address for container-to-container communication
		// This allows toxiproxy to reach floci on the Docker network
		flociInternalAddress := containers["main"].getInternalAddress("main")
		if flociInternalAddress == "" {
			// Fallback to external binding if internal is not available
			flociInternalAddress = endpoint
		}

		if proxy, err = toxiproxyClient.CreateProxy("floci", ":56248", flociInternalAddress); err != nil {
			return nil, fmt.Errorf("can not create toxiproxy proxy for floci component: %w", err)
		}

		clientContainer = containers["toxiproxy"]
		endpoint = clientContainer.bindings["main"].getAddress()
	}

	clientContainers := map[string]*Container{"main": clientContainer}
	dynamoDb, err := new(dynamoDbFactory).Component(config, logger, clientContainers, &dynamoDbSettings{Region: s.Region})
	if err != nil {
		return nil, err
	}
	s3, err := new(s3Factory).Component(config, logger, clientContainers, &s3Settings{Region: s.Region})
	if err != nil {
		return nil, err
	}

	component := &flociComponent{
		config:          config,
		logger:          logger,
		endpointAddress: endpoint,
		region:          s.Region,
		toxiproxy:       proxy,
		dynamodb:        dynamoDb.(*dynamoDbComponent),
		s3:              s3.(*s3Component),
	}

	return component, nil
}
