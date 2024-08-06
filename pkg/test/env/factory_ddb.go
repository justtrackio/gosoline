package env

import (
	"context"
	"fmt"

	toxiproxy "github.com/Shopify/toxiproxy/v2/client"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/ddb"
	"github.com/justtrackio/gosoline/pkg/log"
)

func init() {
	componentFactories[componentDdb] = &ddbFactory{}
}

const componentDdb = "ddb"

type ddbFactory struct {
	toxiproxyFactory toxiproxyFactory
}

func (f *ddbFactory) Detect(config cfg.Config, manager *ComponentsConfigManager) error {
	if !config.IsSet("cloud.aws.dynamodb") {
		return nil
	}

	if manager.HasType(componentDdb) {
		return nil
	}

	settings := &ddbSettings{}
	UnmarshalSettings(config, settings, componentDdb, "default")
	settings.Type = componentDdb

	if err := manager.Add(settings); err != nil {
		return fmt.Errorf("can not add default ddb component: %w", err)
	}

	return nil
}

func (f *ddbFactory) GetSettingsSchema() ComponentBaseSettingsAware {
	return &ddbSettings{}
}

func (f *ddbFactory) DescribeContainers(settings interface{}) componentContainerDescriptions {
	s := settings.(*ddbSettings)

	descriptions := componentContainerDescriptions{
		"main": {
			containerConfig: f.configureContainer(settings),
			healthCheck:     f.healthCheck(),
		},
	}

	if s.ToxiproxyEnabled {
		descriptions["toxiproxy"] = f.toxiproxyFactory.describeContainer(s.ExpireAfter)
	}

	return descriptions
}

func (f *ddbFactory) configureContainer(settings interface{}) *containerConfig {
	s := settings.(*ddbSettings)

	return &containerConfig{
		Repository: s.Image.Repository,
		Tag:        s.Image.Tag,
		PortBindings: portBindings{
			"8000/tcp": s.Port,
		},
		ExpireAfter: s.ExpireAfter,
	}
}

func (f *ddbFactory) healthCheck() ComponentHealthCheck {
	return func(container *container) error {
		var err error
		var client *dynamodb.Client

		if client, err = f.client(container); err != nil {
			return fmt.Errorf("can not build client: %w", err)
		}

		_, err = client.ListTables(context.Background(), &dynamodb.ListTablesInput{})

		return err
	}
}

func (f *ddbFactory) Component(config cfg.Config, logger log.Logger, containers map[string]*container, settings interface{}) (Component, error) {
	s := settings.(*ddbSettings)

	var err error
	var proxy *toxiproxy.Proxy
	ddbAddress := containers["main"].bindings["8000/tcp"].getAddress()
	namingSettings := ddb.GetTableNamingSettings(config, s.Name)

	if s.ToxiproxyEnabled {
		toxiproxyClient := f.toxiproxyFactory.client(containers["toxiproxy"])

		if proxy, err = toxiproxyClient.CreateProxy("ddb", ":56248", ddbAddress); err != nil {
			return nil, fmt.Errorf("can not create toxiproxy proxy for ddb component: %w", err)
		}

		ddbAddress = containers["toxiproxy"].bindings["56248/tcp"].getAddress()
	}

	component := &DdbComponent{
		baseComponent: baseComponent{
			name: s.Name,
		},
		logger:         logger,
		ddbAddress:     ddbAddress,
		namingSettings: namingSettings,
		toxiproxy:      proxy,
	}

	return component, nil
}

func (f *ddbFactory) client(container *container) (*dynamodb.Client, error) {
	binding := container.bindings["8000/tcp"]
	address := fmt.Sprintf("http://%s:%s", binding.host, binding.port)

	var err error
	var cfg aws.Config

	if cfg, err = GetDefaultAwsSdkConfig(address); err != nil {
		return nil, fmt.Errorf("can't get default aws sdk config: %w", err)
	}

	return dynamodb.NewFromConfig(cfg), nil
}
