package env

import (
	"fmt"
	toxiproxy "github.com/Shopify/toxiproxy/client"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/log"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
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
	toxiproxyFactory toxiproxyFactory
}

func (f *ddbFactory) Detect(config cfg.Config, manager *ComponentsConfigManager) error {
	if !config.IsSet("aws_dynamoDb_endpoint") {
		return nil
	}

	if manager.HasType(componentDdb) {
		return nil
	}

	settings := &ddbSettings{}
	config.UnmarshalDefaults(settings)

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
		Repository: "amazon/dynamodb-local",
		Tag:        "1.15.0",
		PortBindings: portBindings{
			"8000/tcp": s.Port,
		},
		ExpireAfter: s.ExpireAfter,
	}
}

func (f *ddbFactory) healthCheck() ComponentHealthCheck {
	return func(container *container) error {
		ddbClient := f.client(container)
		_, err := ddbClient.ListTables(&dynamodb.ListTablesInput{})

		return err
	}
}

func (f *ddbFactory) Component(_ cfg.Config, logger log.Logger, containers map[string]*container, settings interface{}) (Component, error) {
	s := settings.(*ddbSettings)

	var err error
	var proxy *toxiproxy.Proxy
	var ddbAddress = containers["main"].bindings["8000/tcp"].getAddress()

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
		logger:     logger,
		ddbAddress: ddbAddress,
		toxiproxy:  proxy,
	}

	return component, nil
}

func (f *ddbFactory) client(container *container) *dynamodb.DynamoDB {
	binding := container.bindings["8000/tcp"]
	address := fmt.Sprintf("http://%s:%s", binding.host, binding.port)

	sess := session.Must(session.NewSession(&aws.Config{
		Endpoint:    aws.String(address),
		Region:      aws.String("eu-central-1"),
		MaxRetries:  aws.Int(0),
		Credentials: credentials.NewStaticCredentials("id", "secret", "token"),
	}))

	return dynamodb.New(sess)
}
