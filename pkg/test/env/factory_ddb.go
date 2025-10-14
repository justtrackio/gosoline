package env

import (
	"context"
	"fmt"
	"sync"

	toxiproxy "github.com/Shopify/toxiproxy/v2/client"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/justtrackio/gosoline/pkg/cfg"
	gosoAws "github.com/justtrackio/gosoline/pkg/cloud/aws"
	"github.com/justtrackio/gosoline/pkg/ddb"
	"github.com/justtrackio/gosoline/pkg/log"
)

func init() {
	componentFactories[componentDdb] = &ddbFactory{}
}

const componentDdb = "ddb"

type ddbSettings struct {
	ComponentBaseSettings
	ComponentContainerSettings
	Port             int  `cfg:"port" default:"0"`
	ToxiproxyEnabled bool `cfg:"toxiproxy_enabled" default:"false"`
}

type ddbFactory struct {
	lck              sync.Mutex
	clients          map[string]*dynamodb.Client
	toxiproxyFactory toxiproxyFactory
}

func (f *ddbFactory) Detect(config cfg.Config, manager *ComponentsConfigManager) error {
	if !config.IsSet("cloud.aws.dynamodb") {
		return nil
	}

	if has, err := manager.HasType(componentDdb); err != nil {
		return fmt.Errorf("failed to check if component exists: %w", err)
	} else if has {
		return nil
	}

	settings := &ddbSettings{}
	if err := UnmarshalSettings(config, settings, componentDdb, "default"); err != nil {
		return fmt.Errorf("can not parse ddb settings: %w", err)
	}
	settings.Type = componentDdb

	if err := manager.Add(settings); err != nil {
		return fmt.Errorf("can not add default ddb component: %w", err)
	}

	return nil
}

func (f *ddbFactory) GetSettingsSchema() ComponentBaseSettingsAware {
	return &ddbSettings{}
}

func (f *ddbFactory) DescribeContainers(settings any) ComponentContainerDescriptions {
	s := settings.(*ddbSettings)

	descriptions := ComponentContainerDescriptions{
		"main": {
			ContainerConfig: f.configureContainer(settings),
			HealthCheck:     f.healthCheck(),
		},
	}

	if s.ToxiproxyEnabled {
		descriptions["toxiproxy"] = f.toxiproxyFactory.describeContainer()
	}

	return descriptions
}

func (f *ddbFactory) configureContainer(settings any) *ContainerConfig {
	s := settings.(*ddbSettings)

	return &ContainerConfig{
		Auth:       s.Image.Auth,
		Repository: s.Image.Repository,
		Tag:        s.Image.Tag,
		PortBindings: PortBindings{
			"main": {
				ContainerPort: 8000,
				HostPort:      s.Port,
				Protocol:      "tcp",
			},
		},
	}
}

func (f *ddbFactory) healthCheck() ComponentHealthCheck {
	return func(container *Container) error {
		var err error
		var client *dynamodb.Client

		if client, err = f.client(container); err != nil {
			return fmt.Errorf("can not build client: %w", err)
		}

		_, err = client.ListTables(context.Background(), &dynamodb.ListTablesInput{})

		return err
	}
}

func (f *ddbFactory) Component(config cfg.Config, logger log.Logger, containers map[string]*Container, settings any) (Component, error) {
	var err error
	var namingSettings *ddb.TableNamingSettings
	var proxy *toxiproxy.Proxy

	s := settings.(*ddbSettings)
	ddbAddress := containers["main"].bindings["main"].getAddress()

	if namingSettings, err = ddb.GetTableNamingSettings(config, s.Name); err != nil {
		return nil, fmt.Errorf("can not get table naming settings for ddb component: %w", err)
	}

	if s.ToxiproxyEnabled {
		toxiproxyClient := f.toxiproxyFactory.client(containers["toxiproxy"])

		if proxy, err = toxiproxyClient.CreateProxy("ddb", ":56248", ddbAddress); err != nil {
			return nil, fmt.Errorf("can not create toxiproxy proxy for ddb component: %w", err)
		}

		ddbAddress = containers["toxiproxy"].bindings["main"].getAddress()
	}

	component := &DdbComponent{
		baseComponent: baseComponent{
			name: s.Name,
		},
		config:         config,
		logger:         logger,
		ddbAddress:     ddbAddress,
		namingSettings: namingSettings,
		toxiproxy:      proxy,
	}

	return component, nil
}

func (f *ddbFactory) client(container *Container) (*dynamodb.Client, error) {
	binding := container.bindings["main"]
	address := fmt.Sprintf("http://%s:%s", binding.host, binding.port)

	f.lck.Lock()
	defer f.lck.Unlock()

	if f.clients == nil {
		f.clients = make(map[string]*dynamodb.Client)
	}

	if _, ok := f.clients[address]; !ok {
		var err error
		var cfg aws.Config

		if cfg, err = GetDefaultAwsSdkConfig(); err != nil {
			return nil, fmt.Errorf("can't get default aws sdk config: %w", err)
		}

		f.clients[address] = dynamodb.NewFromConfig(cfg, func(options *dynamodb.Options) {
			options.BaseEndpoint = gosoAws.NilIfEmpty(address)
		})
	}

	return f.clients[address], nil
}
