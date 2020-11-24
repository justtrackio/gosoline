package env

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

func init() {
	componentFactories[componentDdb] = new(ddbFactory)
}

const componentDdb = "ddb"

type ddbSettings struct {
	ComponentBaseSettings
	ComponentContainerSettings
	Port int `cfg:"port" default:"0"`
}

type ddbFactory struct {
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

func (f *ddbFactory) ConfigureContainer(settings interface{}) *containerConfig {
	s := settings.(*ddbSettings)

	return &containerConfig{
		Repository: "amazon/dynamodb-local",
		Tag:        "1.13.0",
		PortBindings: portBindings{
			"8000/tcp": s.Port,
		},
		ExpireAfter: s.ExpireAfter,
	}
}

func (f *ddbFactory) HealthCheck(_ interface{}) ComponentHealthCheck {
	return func(container *container) error {
		ddbClient := f.client(container)
		_, err := ddbClient.ListTables(&dynamodb.ListTablesInput{})

		return err
	}
}

func (f *ddbFactory) Component(_ cfg.Config, logger mon.Logger, container *container, _ interface{}) (Component, error) {
	component := &ddbComponent{
		logger:  logger,
		binding: container.bindings["8000/tcp"],
	}

	return component, nil
}

func (f *ddbFactory) client(container *container) *dynamodb.DynamoDB {
	binding := container.bindings["8000/tcp"]
	address := fmt.Sprintf("http://%s:%s", binding.host, binding.port)

	sess := session.Must(session.NewSession(&aws.Config{
		Endpoint:   aws.String(address),
		Region:     aws.String("eu-central-1"),
		MaxRetries: aws.Int(0),
	}))

	return dynamodb.New(sess)
}
